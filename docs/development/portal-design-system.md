# Portal shell and design system

The taxpayer, officer, and administrator portals use `@openrevenue/ui` for
semantic tokens and accessible controls. Shared controls preserve native HTML
behavior, expose visible focus, use 44-pixel minimum targets, and connect
validation errors with `aria-describedby` and live announcements.

## Responsive layout

The baseline is mobile-first:

| Range | Layout |
| --- | --- |
| Below 768px | Single column; primary navigation uses an explicitly labelled menu |
| 768–1023px | Sidebar layout with two-column content cards |
| 1024px and above | Persistent 256px sidebar with three-column summary cards |

Content must reflow at 320 CSS pixels and at 400% zoom without horizontal
scrolling except for intrinsically two-dimensional data tables.

## Reusable states

`/components` documents primary, secondary, loading, success, warning, empty,
and validation-error variants in the running portal. Authentication failures
show a neutral sign-in prompt. Authorization failures render no protected
content or identifiers. Route changes focus the page heading, while the skip
link and landmarks provide predictable keyboard navigation.

## Accessibility verification

Run `pnpm --filter @openrevenue/taxpayer-portal test`. The suite runs Axe
structural rules, keyboard routing, route focus, error announcements,
authorization redaction, authentication fallback, and WCAG AA token contrast
checks. Manual release review still covers browser zoom, high contrast,
screen-reader announcements, localization expansion, and reduced motion.

No telemetry may include form values, taxpayer identifiers, tokens, or
authorization details. Portal failures should be counted by route and stable
error category only.
