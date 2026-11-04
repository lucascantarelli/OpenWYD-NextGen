import type { NavIcon } from "./nav";

/** Ícones em SVG inline (24×24, traço 1.6) — sem dependência externa. */
const PATHS: Record<NavIcon, string> = {
  gauge: "M12 14l4-4M4.5 18a8.5 8.5 0 1 1 15 0",
  package: "M12 3l8 4.5v9L12 21l-8-4.5v-9L12 3zm0 0v18m8-13.5L12 12 4 7.5",
  swords: "M4 20l6-6M14 4l6 6-8 8-4-4 6-6zM14 20l-4-4",
  store: "M4 9h16l-1.5-4.5h-13L4 9zm1 0v11h14V9M9 20v-6h6v6",
  percent: "M6 18L18 6M8 6.5a1.5 1.5 0 1 1-3 0 1.5 1.5 0 0 1 3 0zm11 11a1.5 1.5 0 1 1-3 0 1.5 1.5 0 0 1 3 0z",
  map: "M9 4L3 6.5v13L9 17l6 3 6-2.5v-13L15 7 9 4zm0 0v13m6-10v13",
  sparkles: "M12 3l1.8 4.6L18 9.5l-4.2 1.9L12 16l-1.8-4.6L6 9.5l4.2-1.9L12 3zM19 15l.9 2.3L22 18l-2.1.9L19 21l-.9-2.1L16 18l2.1-.7L19 15z",
  sliders: "M4 7h9m4 0h3M4 17h3m4 0h9M14 4v6M8 14v6",
  users: "M8 11a3 3 0 1 0 0-6 3 3 0 0 0 0 6zm8 0a3 3 0 1 0 0-6 3 3 0 0 0 0 6zM2.5 20c0-3 2.5-5 5.5-5s5.5 2 5.5 5m2.5-5c2.7 0 5 1.9 5 5",
  activity: "M3 12h4l2.5-6 4 12 2.5-6h5",
  receipt: "M6 3h12v18l-3-2-3 2-3-2-3 2V3zm3 5h6M9 12h6",
  tag: "M13.5 3.5H4.5v9L14 22l8-8-8.5-10.5zM8 8h.01",
  compass: "M12 3a9 9 0 1 0 0 18 9 9 0 0 0 0-18zm3.6 5.4l-2.1 5.1-5.1 2.1 2.1-5.1 5.1-2.1z",
  route: "M6 19a2 2 0 1 0 0-4 2 2 0 0 0 0 4zm12-10a2 2 0 1 0 0-4 2 2 0 0 0 0 4zM16 9c0 3.5-2 4.2-4 5.5S8 16 8 17",
  user: "M12 12a4 4 0 1 0 0-8 4 4 0 0 0 0 8zM4 21c0-3.6 3.6-6 8-6s8 2.4 8 6",
  castle: "M4 21V9l3-2V4h3v2h4V4h3v3l3 2v12M9 21v-5h6v5",
  flask: "M9 3h6M10 3v5l-5 9a3 3 0 0 0 2.7 4.5h8.6A3 3 0 0 0 19 17l-5-9V3M7.5 15h9",
  book: "M5 4h9a3 3 0 0 1 3 3v13H8a3 3 0 0 1-3-3V4zm12 3v13",
  terminal: "M3.5 5h17v14h-17zM7 9.5l2.5 2.5L7 14.5M12.5 15H17",
};

export function Icon({ name, size = 16 }: { name: NavIcon; size?: number }) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={1.6}
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
      focusable="false"
    >
      <path d={PATHS[name]} />
    </svg>
  );
}
