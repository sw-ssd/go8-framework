//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/mxschmitt/playwright-go"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"codeberg.org/gmhafiz/go8/internal/server"
)

const (
	apiPort   = "8080"
	webPort   = "3000"
	apiBase   = "http://localhost:" + apiPort // baked into the web build (VITE_API_BASE)
	webOrigin = "http://localhost:" + webPort // CORS allowed origin + page URL
)

var browser playwright.Browser

func TestMain(m *testing.M) {
	ctx := context.Background()

	// --- Postgres via Testcontainers (Docker or Podman) ---
	pg, err := postgres.Run(ctx, "postgres:17",
		postgres.WithDatabase("go8_e2e_db"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "postgres container: %v\n", err)
		os.Exit(1)
	}
	pgHost, _ := pg.Host(ctx)
	pgPort, _ := pg.MappedPort(ctx, "5432/tcp")

	// --- API configuration (read by config.New before server.New) ---
	os.Setenv("NEWAPI_HOST", "0.0.0.0")
	os.Setenv("NEWAPI_PORT", apiPort)
	os.Setenv("NEWAPI_RUN_SWAGGER", "false")
	os.Setenv("DB_DRIVER", "postgres")
	os.Setenv("DB_HOST", pgHost)
	os.Setenv("DB_PORT", pgPort.Port())
	os.Setenv("DB_USER", "postgres")
	os.Setenv("DB_PASS", "postgres")
	os.Setenv("DB_NAME", "go8_e2e_db")
	os.Setenv("DB_SSL_MODE", "disable")
	os.Setenv("CORS_ALLOWED_ORIGINS", webOrigin)

	// --- API in-process (real code path) ---
	s := server.New(server.WithVersion("e2e"))
	s.Init()
	s.Migrate()
	go s.Run()
	waitFor("http://localhost:"+apiPort+"/api/health", 60*time.Second)

	// --- Web: build with baked API base, then preview ---
	_, thisFile, _, _ := runtime.Caller(0)
	webDir := filepath.Join(filepath.Dir(thisFile), "../web")
	build := exec.Command("npm", "run", "build")
	build.Dir = webDir
	build.Env = append(os.Environ(), "VITE_API_BASE="+apiBase)
	if out, berr := build.CombinedOutput(); berr != nil {
		fmt.Fprintf(os.Stderr, "web build failed: %v\n%s\n", berr, out)
		os.Exit(1)
	}
	preview := exec.Command("npx", "vite", "preview", "--host", "--port", webPort, "--strictPort")
	preview.Dir = webDir
	if perr := preview.Start(); perr != nil {
		fmt.Fprintf(os.Stderr, "web preview start: %v\n", perr)
		os.Exit(1)
	}
	waitFor("http://localhost:"+webPort+"/", 60*time.Second)

	// --- Browser: native headless Chromium via Playwright ---
	if ierr := playwright.Install(); ierr != nil {
		fmt.Fprintf(os.Stderr, "playwright install: %v\n", ierr)
		os.Exit(1)
	}
	pw, perr := playwright.Run()
	if perr != nil {
		fmt.Fprintf(os.Stderr, "playwright run: %v\n", perr)
		os.Exit(1)
	}
	browser, perr = pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
	})
	if perr != nil {
		fmt.Fprintf(os.Stderr, "chromium launch: %v\n", perr)
		os.Exit(1)
	}

	code := m.Run()

	// explicit cleanup (os.Exit skips defers)
	if browser != nil {
		_ = browser.Close()
	}
	if preview.Process != nil {
		_ = preview.Process.Kill()
	}
	_ = s.Shutdown(context.Background())
	_ = pg.Terminate(ctx)
	os.Exit(code)
}

func TestTodoCRUD(t *testing.T) {
	require.NotNil(t, browser)

	page, err := browser.NewPage()
	require.NoError(t, err)
	defer page.Close()

	_, err = page.Goto(webOrigin + "/")
	require.NoError(t, err)

	require.NoError(t, page.Fill("input[placeholder=\"New todo\"]", "Buy milk"))
	require.NoError(t, page.Click("button:has-text(\"Add\")"))

	item := page.Locator("ul li:has-text(\"Buy milk\")")
	require.NoError(t, item.First().WaitFor())

	n, err := item.Count()
	require.NoError(t, err)
	require.Equal(t, 1, n, "created todo should be listed")

	require.NoError(t, item.Locator("button").Click())

	require.Eventually(t, func() bool {
		c, cerr := item.Count()
		return cerr == nil && c == 0
	}, 10*time.Second, 200*time.Millisecond, "deleted todo should disappear from the list")
}

func waitFor(url string, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode < 500 {
				return
			}
		}
		time.Sleep(300 * time.Millisecond)
	}
}
