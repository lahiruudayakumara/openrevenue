import { useEffect, useRef, useState, type FormEvent } from "react";
import { NavLink, Route, Routes, useLocation } from "react-router";
import {
  Alert,
  Button,
  EmptyState,
  LoadingState,
  PortalShell,
  TextField,
} from "@openrevenue/ui";
import {
  AuthenticationBoundary,
  AuthorizationBoundary,
  type Session,
} from "./auth";
import { useTranslation } from "./i18n";
import { visibleNavigation } from "./navigation";

const defaultSession: Session = {
  status: "authenticated",
  user: { displayName: "Sam Perera", permissions: ["portal:read"] },
};

function PageFocus() {
  const location = useLocation();
  const heading = useRef<HTMLHeadingElement>(null);
  useEffect(() => {
    heading.current?.focus();
  }, [location.pathname]);
  return (
    <h1 ref={heading} tabIndex={-1}>
      Taxpayer dashboard
    </h1>
  );
}

function Dashboard() {
  return (
    <>
      <PageFocus />
      <p className="lede">
        Review filing obligations, payments, and recent notices.
      </p>
      <div className="card-grid">
        {[
          ["Returns due", "1", "Due 31 August 2026"],
          ["Balance", "LKR 12,500", "Updated today"],
          ["Unread notices", "2", "Review securely"],
        ].map(([label, value, detail]) => (
          <section className="summary-card" key={label}>
            <h2>{label}</h2>
            <strong>{value}</strong>
            <p>{detail}</p>
          </section>
        ))}
      </div>
    </>
  );
}

function PlaceholderPage({ title }: { title: string }) {
  const heading = useRef<HTMLHeadingElement>(null);
  const location = useLocation();
  useEffect(() => heading.current?.focus(), [location.pathname]);
  return (
    <>
      <h1 ref={heading} tabIndex={-1}>
        {title}
      </h1>
      <EmptyState title={`No ${title.toLowerCase()} to show`}>
        <p>This secure area is ready for its domain workflow.</p>
      </EmptyState>
    </>
  );
}

function ComponentExamples() {
  const [reference, setReference] = useState("");
  const [error, setError] = useState("");
  function submit(event: FormEvent) {
    event.preventDefault();
    setError(reference.trim() ? "" : "Enter a payment reference.");
  }
  return (
    <>
      <h1 tabIndex={-1}>Reusable component states</h1>
      <p>
        Examples cover default, loading, empty, success, warning, and error
        states.
      </p>
      <div className="example-stack">
        <div>
          <Button>Primary action</Button>{" "}
          <Button variant="secondary">Secondary action</Button>
        </div>
        <LoadingState label="Loading taxpayer records" />
        <Alert tone="success">Payment saved successfully.</Alert>
        <Alert tone="warning">Your session expires in five minutes.</Alert>
        <form onSubmit={submit} noValidate>
          <TextField
            label="Payment reference"
            hint="Use the reference shown on your receipt."
            error={error}
            value={reference}
            onChange={(event) => setReference(event.target.value)}
          />
          <Button type="submit">Validate example</Button>
        </form>
      </div>
    </>
  );
}

function Portal({
  session,
}: {
  session: Extract<Session, { status: "authenticated" }>;
}) {
  const { t } = useTranslation();
  const items = visibleNavigation(session.user.permissions);
  return (
    <PortalShell
      brand={
        <a
          className="brand"
          href="/"
          aria-label={`${t("productName")} ${t("portalName")} home`}
        >
          <span aria-hidden="true">OR</span>
          <span>
            <strong>{t("productName")}</strong>
            <small>{t("portalName")}</small>
          </span>
        </a>
      }
      account={
        <>
          <span>{session.user.displayName}</span>
          <Button variant="secondary">{t("signOut")}</Button>
        </>
      }
      navigation={(closeMenu) => (
        <>
          {items.map((item) => (
            <NavLink
              end={item.path === "/"}
              key={item.path}
              onClick={closeMenu}
              to={item.path}
            >
              {item.label}
            </NavLink>
          ))}
        </>
      )}
      footer="Fictional demonstration environment · No production taxpayer data"
    >
      <Routes>
        <Route path="/" element={<Dashboard />} />
        {items.slice(1).map((item) => (
          <Route
            key={item.path}
            path={item.path}
            element={<PlaceholderPage title={item.label} />}
          />
        ))}
        <Route path="/components" element={<ComponentExamples />} />
        <Route
          path="/restricted"
          element={
            <AuthorizationBoundary allowed={false}>
              <p>Protected content</p>
            </AuthorizationBoundary>
          }
        />
        <Route path="*" element={<PlaceholderPage title="Page not found" />} />
      </Routes>
    </PortalShell>
  );
}

export function App({ session = defaultSession }: { session?: Session }) {
  return (
    <AuthenticationBoundary session={session}>
      {session.status === "authenticated" && <Portal session={session} />}
    </AuthenticationBoundary>
  );
}
