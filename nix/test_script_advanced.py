# pyright: reportUndefinedVariable=false
# ruff: noqa: F821
import re
import html

COOKIES = "/tmp/cookies.txt"
DUMMY_FILE = "/tmp/dummy.txt"
DUMMY_CONTENT = "bucket-roundtrip-payload"


def curl(args):
    return machine.succeed(f"curl -f {args}").strip()


def status(args):
    return machine.succeed(f"curl -s -o /dev/null -w '%{{http_code}}' {args}").strip()


def bucket_roundtrip():
    machine.succeed(f"printf %s {DUMMY_CONTENT!r} > {DUMMY_FILE}")

    uploaded = curl(
        f"-b {COOKIES} -F 'file=@{DUMMY_FILE}' -F 'plain=true' "
        f"'http://localhost:{PORT}/api/file/upload'"
    ).lstrip("/")
    assert uploaded, "expected upload to return a file name"

    fetched = curl(f"'http://localhost:{PORT}/{uploaded}'")
    assert fetched == DUMMY_CONTENT, (
        f"bucket round-trip mismatch: got {fetched!r}, expected {DUMMY_CONTENT!r}"
    )

    curl(
        f"-b {COOKIES} -X DELETE -F 'file_name={uploaded}' "
        f"'http://localhost:{PORT}/api/account/file'"
    )

    code = status(f"'http://localhost:{PORT}/{uploaded}'")
    assert code == "307", f"expected 307 for deleted file, got {code}"


def submit_dex_login(page):
    m = re.search(r"""<form[^>]*action=["']([^"']+)["']""", page)
    assert m, f"no <form action=...> in dex page (len={len(page)}):\n{page}"
    action = html.unescape(m.group(1))
    if action.startswith("/"):
        action = f"http://localhost:{DEX_PORT}{action}"

    machine.succeed(
        f"curl -fsSL -b {COOKIES} -c {COOKIES} "
        f"--data-urlencode 'login={DEX_EMAIL}' "
        f"--data-urlencode 'password={DEX_PASSWORD}' "
        f"'{action}'"
    )


def oidc_link_flow():
    page = machine.succeed(
        f"curl -fsSL -b {COOKIES} -c {COOKIES} -d '' "
        f"'http://localhost:{PORT}/api/auth/link/openid-connect'"
    )
    submit_dex_login(page)

    code = status(f"-b {COOKIES} 'http://localhost:{PORT}/api/account/files'")
    assert code == "200", f"expected 200 after link, got {code}"


def garage_object_count():
    out = machine.succeed("garage bucket info hostling")
    m = re.search(r"objects?:\s*(\d+)", out, re.IGNORECASE)
    assert m, f"couldn't parse object count from:\n{out}"
    return int(m.group(1))


def verify_account_deletion_purges_bucket():
    invite = machine.succeed(
        f"curl -fsS -b {COOKIES} -F 'id=1' -F 'uses=1' "
        f"'http://localhost:{PORT}/api/admin/give_invite_code'"
    ).strip()

    cookies_b = "/tmp/cookies_b.txt"
    machine.succeed(
        f"curl -f -c {cookies_b} -L --data-urlencode 'code={invite}' "
        f"'http://localhost:{PORT}/api/auth/register'"
    )

    baseline = garage_object_count()

    machine.succeed("head -c 256 /dev/urandom > /tmp/b1.bin")
    machine.succeed("head -c 512 /dev/urandom > /tmp/b2.bin")
    curl(
        f"-b {cookies_b} -F 'file=@/tmp/b1.bin' -F 'plain=true' "
        f"'http://localhost:{PORT}/api/file/upload'"
    )
    curl(
        f"-b {cookies_b} -F 'file=@/tmp/b2.bin' -F 'plain=true' "
        f"'http://localhost:{PORT}/api/file/upload'"
    )

    after_upload = garage_object_count()
    assert after_upload == baseline + 2, (
        f"expected {baseline + 2} objects after 2 uploads, got {after_upload}"
    )

    machine.succeed(
        f"curl -f -b {cookies_b} -X DELETE 'http://localhost:{PORT}/api/account/'"
    )

    after_delete = garage_object_count()
    assert after_delete == baseline, (
        f"account deletion leaked blobs: expected {baseline}, got {after_delete}"
    )


def verify_unlink_last_identity_blocked():
    code = status(
        f"-b {COOKIES} -d '' "
        f"'http://localhost:{PORT}/api/auth/unlink/openid-connect'"
    )
    assert code == "409", f"expected 409 unlinking last identity, got {code}"


def oidc_login_flow():
    # try login logout
    machine.succeed(
        f"curl -f -b {COOKIES} -c {COOKIES} -X POST 'http://localhost:{PORT}/logout'"
    )
    code = status(f"-b {COOKIES} 'http://localhost:{PORT}/api/account/files'")
    assert code == "401", f"expected 401 after logout, got {code}"

    page = machine.succeed(
        f"curl -fsSL -b {COOKIES} -c {COOKIES} "
        f"'http://localhost:{PORT}/api/auth/login/openid-connect'"
    )
    submit_dex_login(page)

    code = status(f"-b {COOKIES} 'http://localhost:{PORT}/api/account/files'")
    assert code == "200", f"expected 200 after OIDC login, got {code}"


def main():
    start_all()
    machine.wait_for_unit("garage-init.service")
    machine.wait_for_unit("dex.service")
    machine.wait_for_unit("hostling.service")
    machine.wait_for_open_port(int(PORT))
    machine.wait_for_open_port(int(DEX_PORT))
    curl(f"'http://localhost:{PORT}/'")

    machine.succeed(
        f"curl -f -c {COOKIES} -L --data-urlencode 'code={TOKEN}' "
        f"'http://localhost:{PORT}/api/auth/register'"
    )

    bucket_roundtrip()
    oidc_link_flow()
    verify_unlink_last_identity_blocked()
    oidc_login_flow()
    verify_account_deletion_purges_bucket()


main()
