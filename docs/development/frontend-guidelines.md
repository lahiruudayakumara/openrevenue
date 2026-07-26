# Frontend guidelines

Organize by feature, keep server state in TanStack Query, validate forms with Zod, and reuse workspace packages. Components render decisions but do not implement tax rules. All controls need accessible names, keyboard support, localization, loading/error states, and tests. Mock APIs with MSW and reserve Playwright for critical journeys.
# Frontend guidelines

Portal UI uses semantic HTML and the shared `@openrevenue/ui` tokens and
controls. New routes must use the authenticated shell, apply permission checks
before rendering protected data, provide loading/empty/error states, and move
focus to the new page heading. All user-visible strings pass through the
localization boundary.

See [Portal shell and design system](portal-design-system.md) for responsive
breakpoints, reusable states, accessibility verification, privacy, and
observability requirements.
