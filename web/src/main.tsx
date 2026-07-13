import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { RouterApp } from "./RouterApp";
import "./styles.css";

const root = document.getElementById("root");

if (!root) {
  throw new Error("AreaFlow dashboard root element is missing");
}

createRoot(root).render(
  <StrictMode>
    <RouterApp />
  </StrictMode>,
);
