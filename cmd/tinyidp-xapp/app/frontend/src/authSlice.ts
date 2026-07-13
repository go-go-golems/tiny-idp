import { createSlice, type PayloadAction } from "@reduxjs/toolkit";

export type AuthStatus = "booting" | "authenticated" | "loggedOut";

export interface AuthState {
  status: AuthStatus;
  userId: string;
  csrfToken: string;
}

const initialState: AuthState = {
  status: "booting",
  userId: "",
  csrfToken: ""
};

const authSlice = createSlice({
  name: "auth",
  initialState,
  reducers: {
    sessionEstablished(state, action: PayloadAction<{ userId: string; csrfToken: string }>) {
      state.status = "authenticated";
      state.userId = action.payload.userId;
      state.csrfToken = action.payload.csrfToken;
    },
    sessionEnded(state) {
      state.status = "loggedOut";
      state.userId = "";
      state.csrfToken = "";
    }
  }
});

export const { sessionEstablished, sessionEnded } = authSlice.actions;
export default authSlice.reducer;
