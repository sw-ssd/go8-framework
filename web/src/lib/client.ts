import { createPromiseClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { TodoService } from "../gen/api/go8/v1/todo_connect.js";

const baseUrl = (import.meta.env.VITE_API_BASE as string) ?? "http://localhost:8080";

const transport = createConnectTransport({ baseUrl });

export const todoClient = createPromiseClient(TodoService, transport);
