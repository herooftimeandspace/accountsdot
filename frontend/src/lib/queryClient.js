import { QueryClient } from "@tanstack/react-query";

/**
 * queryClient centralizes read-only frontend request defaults. DEV pages
 * should fail visibly rather than silently retrying stale mock endpoints, and
 * focus changes should not refetch behind an active persona demo.
 */
export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: false,
      refetchOnWindowFocus: false,
    },
    mutations: {
      retry: false,
    },
  },
});
