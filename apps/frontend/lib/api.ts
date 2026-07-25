import axios, { AxiosError, type InternalAxiosRequestConfig } from "axios";

import { useAuthStore } from "@/stores/auth-store";
import type { TokenResponse } from "@/lib/types";

const apiURL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:3001";

export const api = axios.create({
  baseURL: apiURL,
  timeout: 15_000,
  headers: { "Content-Type": "application/json" },
});

let refreshPromise: Promise<TokenResponse> | null = null;

api.interceptors.request.use((config) => {
  const accessToken = useAuthStore.getState().accessToken;
  if (accessToken) {
    config.headers.Authorization = `Bearer ${accessToken}`;
  }
  return config;
});

type RetryableRequest = InternalAxiosRequestConfig & { _retry?: boolean };

api.interceptors.response.use(
  (response) => response,
  async (error: AxiosError) => {
    const request = error.config as RetryableRequest | undefined;
    const state = useAuthStore.getState();
    const isAuthRoute = request?.url?.startsWith("/auth/");

    if (error.response?.status !== 401 || !request || request._retry || isAuthRoute) {
      return Promise.reject(error);
    }
    if (!state.refreshToken) {
      state.clearSession();
      return Promise.reject(error);
    }

    request._retry = true;
    refreshPromise ??= axios
      .post<TokenResponse>(`${apiURL}/auth/refresh`, {
        refresh_token: state.refreshToken,
      })
      .then(({ data }) => data)
      .finally(() => {
        refreshPromise = null;
      });

    try {
      const tokens = await refreshPromise;
      useAuthStore.getState().setTokens(tokens.access_token, tokens.refresh_token);
      request.headers.Authorization = `Bearer ${tokens.access_token}`;
      return api(request);
    } catch (refreshError) {
      useAuthStore.getState().clearSession();
      return Promise.reject(refreshError);
    }
  },
);

export function apiErrorMessage(error: unknown): string {
  if (axios.isAxiosError<{ error?: string }>(error)) {
    return error.response?.data?.error ?? "Unable to connect to ClipStudio AI.";
  }
  return "Something went wrong. Please try again.";
}
