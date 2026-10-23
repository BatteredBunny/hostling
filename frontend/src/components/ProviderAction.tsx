import { splitProps } from "solid-js";
import type { JSX } from "solid-js";
import { GitBranch, KeyRound, KeySquare } from "lucide-solid";
import { Dynamic } from "solid-js/web";
import styles from "./ProviderAction.module.css";

const providers = new Map<string, { label: string; icon: typeof GitBranch }>([
  ["github", { label: "GitHub", icon: GitBranch }],
  ["openid-connect", { label: "OpenID Connect", icon: KeyRound }],
]);

export function providerPresentation(name: string) {
  return providers.get(name) ?? { label: name, icon: KeySquare };
}

export function providerLabel(name: string) {
  return providerPresentation(name).label;
}

export function ProviderIcon(props: { name: string }) {
  return (
    <Dynamic
      component={providerPresentation(props.name).icon}
      class={`lucide-icon ${styles.icon}`}
      aria-hidden="true"
    />
  );
}

export function ProviderLink(props: {
  provider: string;
  href: string;
  children: JSX.Element;
}) {
  return (
    <a class={styles.action} href={props.href}>
      <ProviderIcon name={props.provider} />
      <span>{props.children}</span>
    </a>
  );
}

export function ProviderButton(
  props: JSX.ButtonHTMLAttributes<HTMLButtonElement> & { provider: string },
) {
  const [local, button] = splitProps(props, ["provider", "children", "class"]);
  return (
    <button {...button} class={`${styles.action} ${local.class ?? ""}`}>
      <ProviderIcon name={local.provider} />
      <span>{local.children}</span>
    </button>
  );
}
