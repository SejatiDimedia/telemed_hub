import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/register")({
  component: RegisterPage,
});

function RegisterPage() {
  return (
    <div className="flex min-h-screen items-center justify-center">
      <h1 className="text-2xl font-bold">Register — Placeholder</h1>
    </div>
  );
}
