import { For, createSignal } from "solid-js";
import { useMutation, useQuery } from "@tanstack/solid-query";
import { todoClient } from "../../lib/client";

export function TodoPage() {
  const [title, setTitle] = createSignal("");

  const todos = useQuery(() => ({
    queryKey: ["todos"],
    queryFn: async () => {
      const res = await todoClient.list({ page: 1, pageSize: 30 });
      return res.todos;
    },
  }));

  const create = useMutation(() => ({
    mutationFn: async (t: string) => todoClient.create({ title: t, priority: 0 }),
    onSuccess: () => todos.refetch(),
  }));

  const remove = useMutation(() => ({
    mutationFn: async (id: bigint) => todoClient.delete({ id }),
    onSuccess: () => todos.refetch(),
  }));

  return (
    <main style={{ "max-width": "40rem", margin: "2rem auto", "font-family": "sans-serif" }}>
      <h1>Todos</h1>

      <form
        onSubmit={(e) => {
          e.preventDefault();
          const t = title();
          if (!t) return;
          create.mutate(t);
          setTitle("");
        }}
      >
        <input
          value={title()}
          onInput={(e) => setTitle(e.currentTarget.value)}
          placeholder="New todo"
        />
        <button type="submit">Add</button>
      </form>

      <ul>
        <For each={todos.data ?? []}>
          {(todo) => (
            <li>
              <span>{todo.title}</span>
              <button onClick={() => remove.mutate(todo.id)}>x</button>
            </li>
          )}
        </For>
      </ul>
    </main>
  );
}
