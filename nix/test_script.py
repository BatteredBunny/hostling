# pyright: reportUndefinedVariable=false
# ruff: noqa: F821
import json

COOKIES_A = "/tmp/cookies_a.txt"
DUMMY_FILE = "/tmp/dummy.txt"
DUMMY_CONTENT = "123dummydata"


def _form_args(fields):
    if isinstance(fields, dict):
        fields = fields.items()
    return " ".join(f"-F '{k}={v}'" for k, v in fields)


def _curl_args(path, *, method=None, fields=None, cookies=COOKIES_A):
    parts = []
    if cookies:
        parts.append(f"-b {cookies}")
    if method and method != "GET":
        parts.append(f"-X {method}")
    if fields:
        parts.append(_form_args(fields))
    parts.append(f"'http://localhost:{PORT}{path}'")
    return " ".join(parts)


def api_call(path, **kw):
    """Run a curl, fail on non-2xx, return body stripped."""
    return machine.succeed(f"curl -f {_curl_args(path, **kw)}").strip()


def api_status(path, **kw):
    """Run a curl and return only the HTTP status code as a string."""
    return machine.succeed(
        f"curl -s -o /dev/null -w '%{{http_code}}' {_curl_args(path, **kw)}"
    ).strip()


def api_get(path, **kw):
    return json.loads(api_call(path, **kw))


def upload_file(*, src=None, cookies=COOKIES_A, fields=None):
    base = [("file", f"@{src or DUMMY_FILE}"), ("plain", "true")]
    if fields:
        base.extend(fields.items() if isinstance(fields, dict) else fields)
    return api_call("/api/file/upload", fields=base, cookies=cookies).lstrip("/")


def delete_file_entry(file_name, cookies=COOKIES_A):
    api_call(
        "/api/account/file",
        method="DELETE",
        fields={"file_name": file_name},
        cookies=cookies,
    )


def wait_for_services():
    start_all()
    if DB_TYPE == "postgresql":
        machine.wait_for_unit("postgresql.service")
    machine.wait_for_unit("hostling.service")
    machine.wait_for_open_port(int(PORT))
    api_call("/", cookies=None)
    machine.succeed(f"printf %s {DUMMY_CONTENT!r} > {DUMMY_FILE}")


def register_initial_user():
    # First registration consumes INITIAL_REGISTER_TOKEN and makes the user admin.
    machine.succeed(
        f"curl -f -c {COOKIES_A} -L --data-urlencode 'code={TOKEN}' "
        f"'http://localhost:{PORT}/api/auth/register'"
    )


def upload_initial_files():
    small_file = upload_file()

    # Verify the uploaded file is accessible and has correct content
    downloaded = api_call(f"/{small_file}", cookies=None)
    expected = machine.succeed(f"cat {DUMMY_FILE}")
    assert downloaded == expected, (
        f"File content mismatch: got {repr(downloaded)}, expected {repr(expected)}"
    )

    machine.succeed("head -c 1024 /dev/urandom > /tmp/larger_dummy.bin")
    large_file = upload_file(src="/tmp/larger_dummy.bin")

    return small_file, large_file


def verify_stats_and_listing():
    stats = api_get("/api/account/files/stats")
    assert stats["count"] == 2, f"Expected 2 files, got {stats['count']}"
    assert stats["size_total"] > 0, "Expected non-zero total size"

    listing = api_get("/api/account/files")
    assert listing["count"] == 2, f"Expected count=2, got {listing['count']}"
    assert len(listing["files"]) == 2, f"Expected 2 files, got {len(listing['files'])}"


def verify_file_deletion(small_file, large_file):
    delete_file_entry(large_file)

    listing = api_get("/api/account/files")
    assert listing["count"] == 1, (
        f"Expected 1 file after deletion, got {listing['count']}"
    )
    assert listing["files"][0]["FileName"] == small_file

    # Deleted file must redirect (must look the same as a private file)
    status = api_status(f"/{large_file}", cookies=None)
    assert status == "307", f"Expected 307 redirect for deleted file, got {status}"

    stats = api_get("/api/account/files/stats")
    assert stats["count"] == 1, (
        f"Expected 1 file in stats after deletion, got {stats['count']}"
    )

    delete_file_entry(small_file)

    listing = api_get("/api/account/files")
    assert listing["count"] == 0, (
        f"Expected 0 files after deleting tagged file, got {listing['count']}"
    )


def verify_upload_token_lifecycle():
    upload_token = api_call(
        "/api/account/upload_token", fields={"nickname": "test-token"}
    )
    assert len(upload_token) == 36, (
        f"Expected UUID upload token, got {repr(upload_token)}"
    )

    token_uploaded = upload_file(cookies=None, fields={"upload_token": upload_token})
    assert token_uploaded, f"Expected upload path, got {repr(token_uploaded)}"

    api_call(
        "/api/account/upload_token",
        method="DELETE",
        fields={"upload_token": upload_token},
    )

    status = api_status(
        "/api/file/upload",
        cookies=None,
        fields=[
            ("file", f"@{DUMMY_FILE}"),
            ("plain", "true"),
            ("upload_token", upload_token),
        ],
    )
    assert status == "401", f"Expected 401 after token deletion, got {status}"

    return token_uploaded


def verify_logout_invalidates_session():
    # -c overwrites the cookie jar with whatever the logout response sets
    machine.succeed(
        f"curl -f -b {COOKIES_A} -c {COOKIES_A} -X POST 'http://localhost:{PORT}/logout'"
    )
    status = api_status("/api/account/files")
    assert status == "401", f"Expected 401 after logout, got {status}"

    # A second POST /logout (now with cleared cookies) must hit the
    # already-logged-out branch and return 303, not 5xx.
    status = machine.succeed(
        f"curl -s -o /dev/null -w '%{{http_code}}' -b {COOKIES_A} -X POST "
        f"'http://localhost:{PORT}/logout'"
    ).strip()
    assert status == "303", f"Expected 303 for unauth logout, got {status}"


def main():
    wait_for_services()
    register_initial_user()

    small_file, large_file = upload_initial_files()
    verify_stats_and_listing()
    verify_file_deletion(small_file, large_file)
    verify_upload_token_lifecycle()
    verify_logout_invalidates_session()


main()
