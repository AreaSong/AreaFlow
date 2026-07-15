import type { ReactNode } from "react";
import { useAuth } from "../context/AuthContext";

export function RequireCapability({ capability, children }: { capability: string; children: ReactNode }) {
  const { allowsCapability } = useAuth();
  return allowsCapability(capability) ? children : null;
}
