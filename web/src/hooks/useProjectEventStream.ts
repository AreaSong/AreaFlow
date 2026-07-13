import { useEffect, useState } from "react";
import type { EventRecord } from "../types";

export type StreamState = "idle" | "connecting" | "connected" | "reconnecting";

export function useProjectEventStream(projectKey: string, initialEvents: EventRecord[]) {
  const [events, setEvents] = useState<EventRecord[]>(initialEvents);
  const [status, setStatus] = useState<StreamState>("idle");

  useEffect(() => setEvents(initialEvents), [initialEvents]);

  useEffect(() => {
    if (!projectKey) {
      setStatus("idle");
      return;
    }
    setStatus("connecting");
    const stream = new EventSource(`/api/v1/projects/${encodeURIComponent(projectKey)}/events/stream`);
    stream.onopen = () => setStatus("connected");
    stream.onerror = () => setStatus("reconnecting");
    const receive = (message: MessageEvent<string>) => {
      try {
        const event = JSON.parse(message.data) as EventRecord;
        setEvents((current) => current.some((item) => item.id === event.id) ? current : [event, ...current].slice(0, 200));
      } catch {
        setStatus("reconnecting");
      }
    };
    stream.onmessage = receive;
    stream.addEventListener("project.import.completed", receive as EventListener);
    stream.addEventListener("project.doctor.completed", receive as EventListener);
    return () => stream.close();
  }, [projectKey]);

  return { events, status };
}
