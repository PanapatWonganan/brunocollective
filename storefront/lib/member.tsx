"use client";

import {
  createContext,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import { MEMBER_TOKEN_KEY, memberMe } from "./api";
import type { MemberProfile } from "./types";

interface MemberContextValue {
  member: MemberProfile | null;
  // False until the stored token has been checked — pages that redirect based
  // on login state should wait for this to avoid a flash.
  ready: boolean;
  signIn: (token: string, member: MemberProfile) => void;
  signOut: () => void;
  setMember: (member: MemberProfile) => void;
}

const MemberContext = createContext<MemberContextValue | null>(null);

export function MemberProvider({ children }: { children: ReactNode }) {
  const [member, setMemberState] = useState<MemberProfile | null>(null);
  const [ready, setReady] = useState(false);

  // Validate the stored token once on mount; a rejected token is stale (e.g.
  // expired after 30 days) and gets cleared.
  useEffect(() => {
    let cancelled = false;
    (async () => {
      if (!localStorage.getItem(MEMBER_TOKEN_KEY)) {
        setReady(true);
        return;
      }
      const profile = await memberMe();
      if (cancelled) return;
      if (profile) {
        setMemberState(profile);
      } else {
        localStorage.removeItem(MEMBER_TOKEN_KEY);
      }
      setReady(true);
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  const value = useMemo<MemberContextValue>(
    () => ({
      member,
      ready,
      signIn: (token, profile) => {
        localStorage.setItem(MEMBER_TOKEN_KEY, token);
        setMemberState(profile);
      },
      signOut: () => {
        localStorage.removeItem(MEMBER_TOKEN_KEY);
        setMemberState(null);
      },
      setMember: setMemberState,
    }),
    [member, ready]
  );

  return <MemberContext.Provider value={value}>{children}</MemberContext.Provider>;
}

export function useMember(): MemberContextValue {
  const ctx = useContext(MemberContext);
  if (!ctx) throw new Error("useMember must be used within MemberProvider");
  return ctx;
}
