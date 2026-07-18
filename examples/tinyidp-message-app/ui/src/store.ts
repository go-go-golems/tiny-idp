import { configureStore } from "@reduxjs/toolkit";
import { messageApi } from "./api";
export const store=configureStore({reducer:{[messageApi.reducerPath]:messageApi.reducer},middleware:(g)=>g().concat(messageApi.middleware)});
export type RootState=ReturnType<typeof store.getState>;
