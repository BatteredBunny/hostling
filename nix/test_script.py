# pyright: reportUndefinedVariable=false
# ruff: noqa: F821

# Detailed API behavior is covered by the Go integration tests. This smoke test
# exercises the packaged service, database provisioning, cookies and local storage
# under both SQLite and PostgreSQL NixOS configurations.
COOKIES = "/tmp/cookies.txt"
DUMMY_FILE = "/tmp/dummy.txt"
DUMMY_CONTENT = "hostling-service-smoke-test"


def main():
    start_all()
    if DB_TYPE == "postgresql":
        machine.wait_for_unit("postgresql.service")
    machine.wait_for_unit("hostling.service")
    machine.wait_for_open_port(int(PORT))
    machine.succeed(f"curl -fsS 'http://localhost:{PORT}/' > /dev/null")

    machine.succeed(
        f"curl -fsS -c {COOKIES} -L --data-urlencode 'code={TOKEN}' "
        f"'http://localhost:{PORT}/api/auth/register' > /dev/null"
    )
    machine.succeed(f"printf %s {DUMMY_CONTENT!r} > {DUMMY_FILE}")
    uploaded = machine.succeed(
        f"curl -fsS -b {COOKIES} -F 'file=@{DUMMY_FILE}' -F 'plain=true' "
        f"'http://localhost:{PORT}/api/file/upload'"
    ).strip().lstrip("/")
    assert uploaded, "expected upload to return a file name"

    downloaded = machine.succeed(f"curl -fsS 'http://localhost:{PORT}/{uploaded}'")
    assert downloaded == DUMMY_CONTENT, (
        f"file content mismatch: got {downloaded!r}, expected {DUMMY_CONTENT!r}"
    )


main()
