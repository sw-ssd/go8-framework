import { render } from "solid-js/web";
import {
  createRootRoute,
  createRoute,
  createRouter,
  RouterProvider,
} from "@tanstack/solid-router";
import { QueryClient, QueryClientProvider } from "@tanstack/solid-query";
import { TodoPage } from "./features/todo/TodoPage";

const rootRoute = createRootRoute({
  component: () => <TodoPage />,
});

const indexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/",
  component: () => <TodoPage />,
});

const routeTree = rootRoute.addChildren([indexRoute]);
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
