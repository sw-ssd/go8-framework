import { For } from "solid-js";
import { useMutation, useQuery } from "@tanstack/solid-query";
import { createPromiseClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";
import { BookService } from "../../gen/api/go8/v1/book_connect.js";

const transport = createConnectTransport({
  baseUrl: (import.meta.env.VITE_API_BASE as string) ?? "http://localhost:8080",
});
const client = createPromiseClient(BookService, transport);

export function BookPage() {
  const items = useQuery(() => ({
    queryKey: ["books"],
    queryFn: async () => {
      const res = await client.list({ page: 1, pageSize: 30 });
      return res.books;
    },
  }));
  const create = useMutation(() => ({
    mutationFn: async () =>
      client.create({


        title: "",





        imageurl: "",



        description: "",



        authorid: 0,


      }),
    onSuccess: () => items.refetch(),
  }));
  const remove = useMutation(() => ({
    mutationFn: async (id: bigint) => client.delete({ id }),
    onSuccess: () => items.refetch(),
  }));

  return (
    <main style={{ "max-width": "40rem", margin: "2rem auto" }}>
      <h1>Books</h1>
      <button onClick={() => create.mutate()}>Add</button>
      <ul>
        <For each={items.data ?? []}>
          {(it) => (
            <li>
              <span>#{it.id.toString()} {JSON.stringify(it)}</span>
              <button onClick={() => remove.mutate(it.id)}>x</button>
            </li>
          )}
        </For>
      </ul>
    </main>
  );
}
