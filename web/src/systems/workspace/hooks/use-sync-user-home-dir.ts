import { useEffect } from "react";

import { useStatus } from "@/systems/status";

import { useUserHomeDirStore } from "./use-user-home-dir-store";

export function useSyncUserHomeDir() {
  const { data } = useStatus();
  const setUserHomeDir = useUserHomeDirStore(state => state.setUserHomeDir);

  useEffect(() => {
    setUserHomeDir(data?.user_home_dir ?? null);
  }, [data?.user_home_dir, setUserHomeDir]);
}
