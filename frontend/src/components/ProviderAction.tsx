import { omit } from "solid-js";
import type { JSX } from "@solidjs/web";
import GitBranch from "@lucide/solid/icons/git-branch";
import KeyRound from "@lucide/solid/icons/key-round";
import KeySquare from "@lucide/solid/icons/key-square";
import styles from "./ProviderAction.module.css";

const providers = new Map<string, { label: string; icon: typeof GitBranch }>([
  ["github", { label: "GitHub", icon: GitBranch }],
  ["openid-connect", { label: "OpenID Connect", icon: KeyRound }],
]);

function providerPresentation(name: string) {
  return providers.get(name) ?? { label: name, icon: KeySquare };
}

export function providerLabel(name: string) {
  return providerPresentation(name).label;
}

export function ProviderIcon(props: { name: string }) {
  // Re-renders when the provider changes, as <Dynamic> would.
  const icon = () => {
    const Icon = providerPresentation(props.name).icon;
    return <Icon class={`lucide-icon ${styles.icon}`} aria-hidden="true" />;
  };
  return <>{icon()}</>;
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
  const button = omit(props, "provider", "children", "class");
  return (
    <button {...button} class={[styles.action, props.class]}>
      <ProviderIcon name={props.provider} />
      <span>{props.children}</span>
    </button>
  );
}
