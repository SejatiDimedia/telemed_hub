import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/")({
  component: HomePage,
});

function HomePage() {
  return (
    <div className="flex min-h-screen items-center justify-center">
      <div className="text-center">
        <h1 className="text-3xl font-bold">TeleMedHub</h1>
        <p className="mt-2 text-gray-600">Platform Telemedicine & Smart Pharmacy</p>
      </div>
    </div>
  );
}
