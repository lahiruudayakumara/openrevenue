import {
  forwardRef,
  useState,
  useId,
  type ButtonHTMLAttributes,
  type HTMLAttributes,
  type InputHTMLAttributes,
  type PropsWithChildren,
  type ReactNode,
} from "react";

export type ButtonVariant = "primary" | "secondary" | "danger";

export function Button({
  children,
  variant = "primary",
  className = "",
  ...props
}: PropsWithChildren<
  ButtonHTMLAttributes<HTMLButtonElement> & { variant?: ButtonVariant }
>) {
  return (
    <button
      className={`or-button or-button--${variant} ${className}`.trim()}
      {...props}
    >
      {children}
    </button>
  );
}

export const TextField = forwardRef<
  HTMLInputElement,
  InputHTMLAttributes<HTMLInputElement> & {
    label: string;
    hint?: string;
    error?: string;
  }
>(function TextField({ label, hint, error, id, ...props }, ref) {
  const generatedId = useId();
  const inputId = id ?? generatedId;
  const hintId = `${inputId}-hint`;
  const errorId = `${inputId}-error`;
  const describedBy = [hint ? hintId : "", error ? errorId : ""]
    .filter(Boolean)
    .join(" ");
  return (
    <div className="or-field">
      <label htmlFor={inputId}>{label}</label>
      {hint && (
        <span className="or-field__hint" id={hintId}>
          {hint}
        </span>
      )}
      <input
        {...props}
        aria-describedby={describedBy || undefined}
        aria-invalid={error ? true : undefined}
        id={inputId}
        ref={ref}
      />
      {error && (
        <span className="or-field__error" id={errorId} role="alert">
          {error}
        </span>
      )}
    </div>
  );
});

export function Alert({
  children,
  tone = "info",
  ...props
}: PropsWithChildren<
  HTMLAttributes<HTMLDivElement> & {
    tone?: "info" | "success" | "warning" | "danger";
  }
>) {
  return (
    <div
      className={`or-alert or-alert--${tone}`}
      role={tone === "danger" ? "alert" : "status"}
      {...props}
    >
      {children}
    </div>
  );
}

export function LoadingState({ label = "Loading" }: { label?: string }) {
  return (
    <div className="or-state" role="status" aria-live="polite">
      <span className="or-spinner" aria-hidden="true" />
      <span>{label}</span>
    </div>
  );
}

export function EmptyState({
  title,
  children,
  action,
}: PropsWithChildren<{ title: string; action?: ReactNode }>) {
  return (
    <section className="or-empty">
      <h2>{title}</h2>
      <div>{children}</div>
      {action}
    </section>
  );
}

export function SkipLink({ href = "#main-content" }: { href?: string }) {
  return (
    <a className="or-skip-link" href={href}>
      Skip to main content
    </a>
  );
}

export function PortalShell({
  brand,
  account,
  navigation,
  children,
  footer,
}: PropsWithChildren<{
  brand: ReactNode;
  account: ReactNode;
  navigation: (closeMenu: () => void) => ReactNode;
  footer: ReactNode;
}>) {
  const [menuOpen, setMenuOpen] = useState(false);
  const closeMenu = () => setMenuOpen(false);
  return (
    <div className="portal">
      <SkipLink />
      <header className="topbar">
        {brand}
        <div className="account">{account}</div>
        <Button
          className="menu-button"
          variant="secondary"
          aria-expanded={menuOpen}
          aria-controls="primary-navigation"
          onClick={() => setMenuOpen((value) => !value)}
        >
          Menu
        </Button>
      </header>
      <div className="portal-grid">
        <nav
          aria-label="Primary"
          className={menuOpen ? "sidebar sidebar--open" : "sidebar"}
          id="primary-navigation"
        >
          {navigation(closeMenu)}
        </nav>
        <main id="main-content" tabIndex={-1}>
          {children}
        </main>
      </div>
      <footer>{footer}</footer>
    </div>
  );
}
