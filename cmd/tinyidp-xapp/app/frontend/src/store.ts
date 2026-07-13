import { configureStore } from "@reduxjs/toolkit";
import { useDispatch, useSelector } from "react-redux";
import authReducer from "./authSlice";
import { bbsApi } from "./api";

export const store = configureStore({
  reducer: {
    auth: authReducer,
    [bbsApi.reducerPath]: bbsApi.reducer
  },
  middleware: (getDefaultMiddleware) => getDefaultMiddleware().concat(bbsApi.middleware)
});

export type RootState = ReturnType<typeof store.getState>;
export type AppDispatch = typeof store.dispatch;

export const useAppDispatch = useDispatch.withTypes<AppDispatch>();
export const useAppSelector = useSelector.withTypes<RootState>();
