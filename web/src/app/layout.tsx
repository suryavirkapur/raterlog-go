import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "Raterlog",
  description: "Realtime monitoring for product teams"
};

export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
