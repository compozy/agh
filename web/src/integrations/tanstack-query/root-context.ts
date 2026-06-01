import { QueryClient } from "@tanstack/react-query";

function getContext() {
  const queryClient = new QueryClient();
  return {
    queryClient,
  };
}

export { getContext };
