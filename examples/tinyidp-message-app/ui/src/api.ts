import { createApi, fetchBaseQuery } from "@reduxjs/toolkit/query/react";

export type Session = { authenticated: boolean; subject?: string; displayName?: string; csrfToken?: string; registrationEnabled?: boolean; providerRegistrationEnabled?: boolean };
export type Message = { id: number; authorName: string; body: string; createdAt: string };
export type Feed = { messages: Message[]; nextCursor?: string };
export type Registration = { csrfToken: string };
export type Logout = { endSessionUrl?: string };

export const messageApi = createApi({
  reducerPath: "messageApi",
  baseQuery: fetchBaseQuery({ baseUrl: "/" }),
  tagTypes: ["Feed", "Session"],
  endpoints: (builder) => ({
    session: builder.query<Session, void>({ query: () => "api/session", providesTags: ["Session"] }),
    feed: builder.query<Feed, void>({ query: () => "api/messages", providesTags: ["Feed"] }),
    registration: builder.query<Registration, void>({ query: () => "api/registration" }),
    createAccount: builder.mutation<{ next: string }, { body: unknown; csrf: string }>({
      query: ({ body, csrf }) => ({ url: "api/accounts", method: "POST", body, headers: { "X-CSRF-Token": csrf } }),
    }),
    createMessage: builder.mutation<Message, { body: string; csrf: string }>({
      query: ({ body, csrf }) => ({ url: "api/messages", method: "POST", body: { body }, headers: { "X-CSRF-Token": csrf } }),
      invalidatesTags: ["Feed"],
    }),
    localLogout: builder.mutation<void, { csrf: string }>({
      query: ({ csrf }) => ({ url: "auth/logout/local", method: "POST", headers: { "X-CSRF-Token": csrf } }),
      invalidatesTags: ["Session", "Feed"],
    }),
    logout: builder.mutation<Logout, { csrf: string }>({
      query: ({ csrf }) => ({ url: "auth/logout", method: "POST", headers: { "X-CSRF-Token": csrf } }),
      invalidatesTags: ["Session", "Feed"],
    }),
  }),
});

export const {
  useSessionQuery,
  useFeedQuery,
  useRegistrationQuery,
  useCreateAccountMutation,
  useCreateMessageMutation,
  useLocalLogoutMutation,
  useLogoutMutation,
} = messageApi;
