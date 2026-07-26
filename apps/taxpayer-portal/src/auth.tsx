import type { PropsWithChildren } from "react";
import { Alert, Button } from "@openrevenue/ui";

export type Session =
  | { status: "loading" }
  | { status: "anonymous" }
  | {
      status: "authenticated";
      user: { displayName: string; permissions: readonly string[] };
    };

export function AuthenticationBoundary({
  session,
  children,
}: PropsWithChildren<{ session: Session }>) {
  if (session.status === "loading") {
    return <p role="status">Checking your secure session…</p>;
  }
  if (session.status === "anonymous") {
    return (
      <main className="auth-page" id="main-content">
        <h1>Sign in required</h1>
        <p>
          Your session may have expired. Sign in again to continue securely.
        </p>
        <Button>Sign in</Button>
      </main>
    );
  }
  return children;
}

export function AuthorizationBoundary({
  allowed,
  children,
}: PropsWithChildren<{ allowed: boolean }>) {
  if (!allowed) {
    return (
      <Alert tone="danger">
        <h1>Access unavailable</h1>
        <p>
          Your account does not have permission to view this page. No protected
          information has been displayed.
        </p>
      </Alert>
    );
  }
  return children;
}
