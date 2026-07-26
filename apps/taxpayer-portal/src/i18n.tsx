import { createContext, useContext, type PropsWithChildren } from "react";

const messages = {
  en: {
    productName: "OpenRevenue",
    portalName: "Taxpayer portal",
    signOut: "Sign out",
  },
  si: {
    productName: "OpenRevenue",
    portalName: "බදු ගෙවන්නන්ගේ ද්වාරය",
    signOut: "ඉවත් වන්න",
  },
} as const;

type Locale = keyof typeof messages;
const LocaleContext = createContext<Locale>("en");

export function LocaleProvider({
  locale = "en",
  children,
}: PropsWithChildren<{ locale?: Locale }>) {
  return (
    <LocaleContext.Provider value={locale}>{children}</LocaleContext.Provider>
  );
}

export function useTranslation() {
  const locale = useContext(LocaleContext);
  return {
    locale,
    t: (key: keyof (typeof messages)["en"]) => messages[locale][key],
  };
}
