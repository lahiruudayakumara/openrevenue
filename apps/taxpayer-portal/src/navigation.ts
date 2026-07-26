export type NavigationItem = {
  label: string;
  path: string;
  requiredPermission?: string;
};

export const taxpayerNavigation: readonly NavigationItem[] = [
  { label: "Dashboard", path: "/" },
  { label: "Profile", path: "/profile" },
  { label: "Tax registrations", path: "/registrations" },
  { label: "Tax returns", path: "/returns" },
  { label: "Payments", path: "/payments" },
  { label: "Ledger", path: "/ledger" },
  { label: "Documents", path: "/documents" },
  { label: "Notifications", path: "/notifications" },
  { label: "Representatives", path: "/representatives" },
  { label: "Account settings", path: "/settings" },
] as const;

export const visibleNavigation = (
  permissions: readonly string[],
): readonly NavigationItem[] =>
  taxpayerNavigation.filter(
    (item) =>
      !item.requiredPermission || permissions.includes(item.requiredPermission),
  );
