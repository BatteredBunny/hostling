import type {
  FullResult,
  Reporter,
  TestCase,
  TestError,
  TestResult,
} from "@playwright/test/reporter";
import { copyFileSync, mkdirSync, readdirSync, rmSync, writeFileSync } from "node:fs";
import { dirname, join, relative } from "node:path";

// Used only when comparing against SCREENSHOT_BASELINE. Sorts every shot into
// unchanged/changed/added/removed and copies the images worth reviewing into
// screenshot-changes/ with a changes.json index for the PR comment.
// Screenshot differences don't fail the run; any other failure still does.

type Status = "unchanged" | "changed" | "added" | "removed";
interface Shot {
  name: string;
  title: string;
  category: string;
  status: Status;
  before?: string;
  after?: string;
  diff?: string;
}
type Attachment = TestResult["attachments"][number];

const output = "screenshot-changes";
const ansi = /\u001b\[[0-9;]*m/g;

export default class ChangesReporter implements Reporter {
  private readonly baseline: string;
  private readonly shots = new Map<string, Shot>();
  private readonly failures: string[] = [];

  constructor(options: { baseline: string }) {
    this.baseline = options.baseline;
  }

  onBegin() {
    rmSync(output, { recursive: true, force: true });
    mkdirSync(output);
  }

  onTestEnd(test: TestCase, result: TestResult) {
    if (result.status === "skipped") return;
    const title = test.titlePath().slice(1).join(" › ");
    const notes = result.annotations;
    let changed = 0;
    notes.forEach((note, index) => {
      if (note.type !== "screenshot" || !note.description) return;
      const name = note.description;
      if (this.shots.has(name)) {
        this.failures.push(`${title}: duplicate screenshot name ${name}`);
        return;
      }
      const find = (suffix: string) =>
        result.attachments.find((attachment) => attachment.name === `${name}-${suffix}.png`);
      const expected = find("expected");
      const actual = find("actual");
      const diff = find("diff");
      const shot: Shot = {
        name,
        title: notes[index + 1]?.type === "title" ? (notes[index + 1].description ?? name) : name,
        category: name.split("/")[0],
        status: expected && actual ? "changed" : actual ? "added" : "unchanged",
      };
      if (shot.status === "changed") {
        changed++;
        shot.before = this.save(name, "before", expected!);
        shot.diff = diff && this.save(name, "diff", diff);
      }
      if (actual) shot.after = this.save(name, "after", actual);
      this.shots.set(name, shot);
    });

    // Every error must be one of this test's screenshot mismatches. Anything else
    // (a failed assertion, a timeout, an unstable screenshot) is a real failure.
    const visual = result.errors.filter((error) =>
      /toHaveScreenshot\(/.test((error.message ?? "").replace(ansi, "")),
    );
    if (
      !["passed", "failed"].includes(result.status) ||
      visual.length !== result.errors.length ||
      visual.length !== changed
    ) {
      const reason = result.errors[0]?.message?.replace(ansi, "").split("\n")[0] ?? result.status;
      this.failures.push(`${title}: ${reason}`);
    }
  }

  // Errors outside any test, e.g. in global setup or the web server.
  onError(error: TestError) {
    this.failures.push((error.message ?? "Unknown error").replace(ansi, "").split("\n")[0]);
  }

  async onEnd(result: FullResult): Promise<{ status: FullResult["status"] }> {
    for (const name of pngs(this.baseline)) {
      if (this.shots.has(name)) continue;
      this.shots.set(name, {
        name,
        title: name,
        category: name.split("/")[0],
        status: "removed",
        before: this.save(name, "before", { path: join(this.baseline, `${name}.png`) }),
      });
    }

    const shots = [...this.shots.values()].sort((a, b) => a.name.localeCompare(b.name));
    const counts = { changed: 0, added: 0, removed: 0, unchanged: 0 };
    for (const shot of shots) counts[shot.status]++;
    writeFileSync(
      join(output, "changes.json"),
      JSON.stringify(
        { counts, shots: shots.filter((shot) => shot.status !== "unchanged") },
        null,
        2,
      ) + "\n",
    );
    console.log(
      `\nScreenshots: ${Object.entries(counts)
        .map(([status, count]) => `${count} ${status}`)
        .join(", ")}`,
    );

    if (this.failures.length || result.status === "interrupted" || result.status === "timedout") {
      console.error(`\nFailures unrelated to visual changes:\n${this.failures.join("\n")}`);
      return { status: result.status === "passed" ? "failed" : result.status };
    }
    return { status: "passed" };
  }

  printsToStdio() {
    return false;
  }

  // Copies one image into screenshot-changes/ and returns its path there.
  private save(name: string, kind: string, attachment: Pick<Attachment, "path" | "body">) {
    const path = join(name, `${kind}.png`);
    mkdirSync(dirname(join(output, path)), { recursive: true });
    if (attachment.body) writeFileSync(join(output, path), attachment.body);
    else if (attachment.path) copyFileSync(attachment.path, join(output, path));
    else throw new Error(`Screenshot attachment without data: ${name} ${kind}`);
    return path;
  }
}

// Shot names in a capture directory: PNG paths relative to it, minus ".png".
function pngs(root: string): string[] {
  return readdirSync(root, { recursive: true, withFileTypes: true })
    .filter((entry) => entry.isFile() && entry.name.endsWith(".png"))
    .map((entry) => relative(root, join(entry.parentPath, entry.name)).slice(0, -4));
}
