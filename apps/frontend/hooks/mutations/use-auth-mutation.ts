import { useMutation } from "@tanstack/react-query";

import { api } from "@/lib/api";
import type { AuthResponse } from "@/lib/types";

export type AuthFormValues = {
  email: string;
  password: string;
  confirmPassword?: string;
};

export function useAuthMutation(mode: "signin" | "signup") {
  return useMutation({
    mutationFn: async ({ email, password }: AuthFormValues) => {
      const endpoint = mode === "signup" ? "/auth/register" : "/auth/login";
      const { data } = await api.post<AuthResponse>(endpoint, { email, password });
      return data;
    },
  });
}
