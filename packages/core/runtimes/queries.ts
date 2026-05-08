import { queryOptions } from "@tanstack/react-query";
import { api } from "../api";

export const runtimeKeys = {
  all: (wsId: string) => ["runtimes", wsId] as const,
  list: (wsId: string) => [...runtimeKeys.all(wsId), "list"] as const,
  listMine: (wsId: string) => [...runtimeKeys.all(wsId), "list", "mine"] as const,
  usageByWorkspace: (wsId: string, days: number) =>
    [...runtimeKeys.all(wsId), "usage", days] as const,
  usage: (rid: string, days: number) =>
    ["runtimes", "usage", rid, days] as const,
  usageByAgent: (rid: string, days: number) =>
    ["runtimes", "usage", "by-agent", rid, days] as const,
  usageByHour: (rid: string, days: number) =>
    ["runtimes", "usage", "by-hour", rid, days] as const,
  latestVersion: () => ["runtimes", "latestVersion"] as const,
};

// Per-runtime usage for detail views. The runtimes list uses the workspace
// batch query below to avoid one aggregate query per visible row.
export function runtimeUsageOptions(runtimeId: string, days: number) {
  return queryOptions({
    queryKey: runtimeKeys.usage(runtimeId, days),
    queryFn: () => api.getRuntimeUsage(runtimeId, { days }),
    staleTime: 60 * 1000,
  });
}

// Batched usage for the runtimes list. One workspace-level aggregate replaces
// one per-row aggregate query while preserving the same RuntimeUsage row shape.
export function runtimeUsageByWorkspaceOptions(wsId: string, days: number) {
  return queryOptions({
    queryKey: runtimeKeys.usageByWorkspace(wsId, days),
    queryFn: () => api.getRuntimesUsage({ workspace_id: wsId, days }),
    staleTime: 60 * 1000,
  });
}

// Per-agent token totals for one runtime — drives the "Cost by agent" tab
// on the runtime detail page. Server-side aggregation keeps the response
// small (one row per agent) regardless of task volume.
export function runtimeUsageByAgentOptions(runtimeId: string, days: number) {
  return queryOptions({
    queryKey: runtimeKeys.usageByAgent(runtimeId, days),
    queryFn: () => api.getRuntimeUsageByAgent(runtimeId, { days }),
    staleTime: 60 * 1000,
  });
}

// Hourly (0..23) token totals for one runtime — drives the "By hour" tab.
export function runtimeUsageByHourOptions(runtimeId: string, days: number) {
  return queryOptions({
    queryKey: runtimeKeys.usageByHour(runtimeId, days),
    queryFn: () => api.getRuntimeUsageByHour(runtimeId, { days }),
    staleTime: 60 * 1000,
  });
}

export function runtimeListOptions(wsId: string, owner?: "me") {
  return queryOptions({
    queryKey: owner === "me" ? runtimeKeys.listMine(wsId) : runtimeKeys.list(wsId),
    queryFn: () => api.listRuntimes({ workspace_id: wsId, owner }),
  });
}

const GITHUB_RELEASES_URL =
  "https://api.github.com/repos/multica-ai/multica/releases/latest";

export function latestCliVersionOptions() {
  return queryOptions({
    queryKey: runtimeKeys.latestVersion(),
    queryFn: async (): Promise<string | null> => {
      try {
        const resp = await fetch(GITHUB_RELEASES_URL, {
          headers: { Accept: "application/vnd.github+json" },
        });
        if (!resp.ok) return null;
        const data = await resp.json();
        return (data.tag_name as string) ?? null;
      } catch {
        return null;
      }
    },
    staleTime: 10 * 60 * 1000, // 10 minutes
  });
}
