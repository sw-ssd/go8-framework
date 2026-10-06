import { render } from "solid-js/web";
import {
  createRootRoute,
  createRoute,
  createRouter,
  Link,
  Outlet,
  RouterProvider,
} from "@tanstack/solid-router";
import { QueryClient, QueryClientProvider } from "@tanstack/solid-query";
import { TodoPage } from "./features/todo/TodoPage";
import { AuthorPage } from "./features/author/AuthorPage";
import { BookPage } from "./features/book/BookPage";

const rootRoute = createRootRoute({
  component: () => (
    <div>
      <nav style={{ padding: "1rem", display: "flex", gap: "1rem" }}>
        <Link href="/">Todos</Link>
        <Link href="/authors">Authors</Link>
        <Link href="/books">Books</Link>
      </nav>
      <Outlet />
    </div>
  ),
});

const indexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/",
  component: () => <TodoPage />,
});

const authorsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/authors",
  component: () => <AuthorPage />,
});

const booksRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/books",
  component: () => <BookPage />,
});

const routeTree = rootRoute.addChildren([indexRoute, authorsRoute, booksRoute]);
const router = createRouter({ routeTree });
const queryClient = new QueryClient();

render(
  () => (
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  ),
  document.getElementById("root")!,
);
