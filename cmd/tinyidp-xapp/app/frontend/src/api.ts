import {
  createApi,
  fetchBaseQuery,
  type FetchBaseQueryError
} from "@reduxjs/toolkit/query/react";
import type { SerializedError } from "@reduxjs/toolkit";
import type { AuthState } from "./authSlice";

export type BBSCategory = "general" | "projects" | "questions" | "notes";

export interface SessionInfo {
  userId: string;
  csrfToken: string;
  tenantIds: string[];
}

export interface CurrentUser {
  id: string;
  kind: string;
  claims: Record<string, unknown>;
}

export interface BBSReply {
  id: string;
  body: string;
  author: string;
  createdAt: string;
}

export interface BBSPost {
  id: string;
  title: string;
  body: string;
  category: BBSCategory;
  author: string;
  createdAt: string;
  canDelete: boolean;
  replies: BBSReply[];
}

export interface BBSBoard {
  name: string;
  description: string;
  posts: BBSPost[];
  stats: { posts: number; replies: number };
}

export const bbsApi = createApi({
  reducerPath: "bbsApi",
  baseQuery: fetchBaseQuery({
    baseUrl: "/",
    prepareHeaders: (headers, { getState, arg }) => {
      const state = getState() as { auth: AuthState };
      const method = typeof arg === "string" ? "GET" : String(arg.method || "GET").toUpperCase();
      if (method !== "GET" && method !== "HEAD" && state.auth.csrfToken) {
        headers.set("X-CSRF-Token", state.auth.csrfToken);
      }
      return headers;
    }
  }),
  tagTypes: ["Board"],
  endpoints: (build) => ({
    getSession: build.query<SessionInfo, void>({
      query: () => "auth/session"
    }),
    getMe: build.query<CurrentUser, void>({
      query: () => "api/me"
    }),
    getBoard: build.query<BBSBoard, void>({
      query: () => "api/bbs",
      providesTags: ["Board"]
    }),
    createPost: build.mutation<BBSBoard, { title: string; body: string; category: BBSCategory }>({
      query: (body) => ({ url: "api/bbs/posts", method: "POST", body }),
      invalidatesTags: ["Board"]
    }),
    createReply: build.mutation<BBSBoard, { postId: string; body: string }>({
      query: ({ postId, body }) => ({
        url: `api/bbs/posts/${encodeURIComponent(postId)}/replies`,
        method: "POST",
        body: { body }
      }),
      invalidatesTags: ["Board"]
    }),
    deletePost: build.mutation<BBSBoard, string>({
      query: (postId) => ({
        url: `api/bbs/posts/${encodeURIComponent(postId)}`,
        method: "DELETE"
      }),
      invalidatesTags: ["Board"]
    }),
    logout: build.mutation<void, void>({
      query: () => ({ url: "auth/logout", method: "POST" })
    })
  })
});

export const {
  useGetSessionQuery,
  useGetMeQuery,
  useGetBoardQuery,
  useCreatePostMutation,
  useCreateReplyMutation,
  useDeletePostMutation,
  useLogoutMutation
} = bbsApi;

export function isUnauthorized(error: FetchBaseQueryError | SerializedError | undefined): boolean {
  if (!error || !("status" in error)) return false;
  if (error.status === 401) return true;
  return error.status === "PARSING_ERROR" && error.originalStatus === 401;
}

export function apiErrorMessage(error: FetchBaseQueryError | SerializedError | undefined): string {
  if (!error) return "The request failed.";
  if ("status" in error) {
    if (typeof error.data === "object" && error.data && "error" in error.data) {
      return String((error.data as { error: unknown }).error).replaceAll("_", " ");
    }
    return `Request failed (${String(error.status)}).`;
  }
  return error.message || "The request failed.";
}
