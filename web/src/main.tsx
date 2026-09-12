import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import AppRouter from "./router";
import { registerDriverSW } from "./offline/register";
import "./styles.css";

registerDriverSW();
createRoot(document.getElementById("root")!).render(<StrictMode><AppRouter /></StrictMode>);
