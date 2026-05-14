# pyright: reportUndefinedVariable=false
# ruff: noqa: F821
import json
import time

COOKIES_A = "/tmp/cookies_a.txt"
COOKIES_B = "/tmp/cookies_b.txt"
COOKIES_THROWAWAY = "/tmp/cookies_throwaway.txt"
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


def add_tag(file_name, tag, cookies=COOKIES_A):
    api_call(
        "/api/account/file/tag",
        fields={"file_name": file_name, "tag": tag},
        cookies=cookies,
    )


def remove_tag(file_name, tag, cookies=COOKIES_A):
    api_call(
        "/api/account/file/tag",
        method="DELETE",
        fields={"file_name": file_name, "tag": tag},
        cookies=cookies,
    )


def toggle_public(file_name, cookies=COOKIES_A):
    api_call(
        "/api/account/file/public",
        fields={"file_name": file_name},
        cookies=cookies,
    )


def delete_file_entry(file_name, cookies=COOKIES_A):
    api_call(
        "/api/account/file",
        method="DELETE",
        fields={"file_name": file_name},
        cookies=cookies,
    )


def admin_give_invite_code(account_id, uses=5):
    return api_call(
        "/api/admin/give_invite_code",
        fields={"id": account_id, "uses": uses},
    )


def admin_delete_account(account_id):
    api_call("/api/admin/user", method="DELETE", fields={"id": account_id})


def admin_delete_files(account_id):
    api_call("/api/admin/files", method="DELETE", fields={"id": account_id})


def admin_delete_sessions(account_id):
    api_call("/api/admin/sessions", method="DELETE", fields={"id": account_id})


def admin_delete_upload_tokens(account_id):
    api_call("/api/admin/upload_tokens", method="DELETE", fields={"id": account_id})


def register_with_invite_code(code, cookies):
    machine.succeed(
        f"curl -f -c {cookies} -L --data-urlencode 'code={code}' "
        f"'http://localhost:{PORT}/api/auth/register'"
    )


def register_status(code, cookies=COOKIES_THROWAWAY):
    return machine.succeed(
        f"curl -s -o /dev/null -w '%{{http_code}}' -c {cookies} "
        f"--data-urlencode 'code={code}' "
        f"'http://localhost:{PORT}/api/auth/register'"
    ).strip()


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


def verify_sorting(small_file, large_file):
    cases = [
        (
            "/api/account/files?sort=file_size&desc=true",
            [large_file, small_file],
            "file_size desc",
        ),
        (
            "/api/account/files?sort=file_size&desc=false",
            [small_file, large_file],
            "file_size asc",
        ),
        (
            "/api/account/files?sort=created_at&desc=true",
            [large_file, small_file],
            "created_at desc",
        ),
        (
            "/api/account/files?sort=created_at&desc=false",
            [small_file, large_file],
            "created_at asc",
        ),
    ]
    for path, expected_order, label in cases:
        listing = api_get(path)
        actual = [f["FileName"] for f in listing["files"]]
        assert actual == expected_order, (
            f"Wrong order for {label}: got {actual}, expected {expected_order}"
        )


def verify_tagging(small_file, large_file):
    add_tag(small_file, "alpha")
    add_tag(small_file, "beta")
    add_tag(large_file, "alpha")

    stats = api_get("/api/account/files/stats")
    assert sorted(stats["tags"]) == ["alpha", "beta"], (
        f"Expected tags [alpha, beta], got {stats['tags']}"
    )

    alpha_files = api_get("/api/account/files?tag=alpha")
    assert alpha_files["count"] == 2, (
        f"Expected 2 files with tag alpha, got {alpha_files['count']}"
    )

    beta_files = api_get("/api/account/files?tag=beta")
    assert beta_files["count"] == 1, (
        f"Expected 1 file with tag beta, got {beta_files['count']}"
    )
    assert beta_files["files"][0]["FileName"] == small_file

    # large_file has alpha, small_file has alpha+beta, so none are untagged
    untagged = api_get("/api/account/files?filter=untagged")
    assert untagged["count"] == 0, f"Expected 0 untagged files, got {untagged['count']}"


def verify_public_private_toggle(small_file, large_file):
    public_files = api_get("/api/account/files?filter=public")
    assert public_files["count"] == 2, (
        f"Expected 2 public files, got {public_files['count']}"
    )

    toggle_public(small_file)

    private_files = api_get("/api/account/files?filter=private")
    assert private_files["count"] == 1, (
        f"Expected 1 private file, got {private_files['count']}"
    )
    assert private_files["files"][0]["FileName"] == small_file

    public_files = api_get("/api/account/files?filter=public")
    assert public_files["count"] == 1, (
        f"Expected 1 public file after toggle, got {public_files['count']}"
    )
    assert public_files["files"][0]["FileName"] == large_file


def verify_tag_removal(small_file, large_file):
    remove_tag(small_file, "beta")

    stats = api_get("/api/account/files/stats")
    assert stats["tags"] == ["alpha"], (
        f"Expected only [alpha] after deleting beta, got {stats['tags']}"
    )

    # small_file still has alpha, so untagged should still be 0
    untagged = api_get("/api/account/files?filter=untagged")
    assert untagged["count"] == 0, f"Expected 0 untagged files, got {untagged['count']}"

    remove_tag(large_file, "alpha")

    untagged = api_get("/api/account/files?filter=untagged")
    assert untagged["count"] == 1, f"Expected 1 untagged file, got {untagged['count']}"
    assert untagged["files"][0]["FileName"] == large_file


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


def verify_tag_too_long_rejected():
    tagged_file = upload_file()
    long_tag = "x" * 26
    status = api_status(
        "/api/account/file/tag",
        fields={"file_name": tagged_file, "tag": long_tag},
    )
    assert status == "400", f"Expected 400 for tag too long, got {status}"


def verify_past_expiry_rejected():
    status = api_status(
        "/api/file/upload",
        fields=[
            ("file", f"@{DUMMY_FILE}"),
            ("plain", "true"),
            ("expiry_timestamp", "1"),
        ],
    )
    assert status == "400", f"Expected 400 for past expiry timestamp, got {status}"


def verify_malformed_expiry_date_rejected():
    # The YYYY-MM-DD parser must reject anything time.Parse can't handle.
    for bad in ["2099-13-99", "not-a-date", "99/12/31", "2099-12"]:
        status = api_status(
            "/api/file/upload",
            fields=[
                ("file", f"@{DUMMY_FILE}"),
                ("plain", "true"),
                ("expiry_date", bad),
            ],
        )
        assert status == "400", f"Expected 400 for expiry_date {bad!r}, got {status}"


def verify_expiry_too_far_rejected():
    # 200 years is way too big lol
    far_future = int(time.time()) + 200 * 365 * 86400
    status = api_status(
        "/api/file/upload",
        fields=[
            ("file", f"@{DUMMY_FILE}"),
            ("plain", "true"),
            ("expiry_timestamp", str(far_future)),
        ],
    )
    assert status == "400", f"Expected 400 for expiry too far in the future, got {status}"


def verify_too_many_tags_rejected():
    fields = [("file", f"@{DUMMY_FILE}"), ("plain", "true")]
    fields.extend(("tag", f"t{i}") for i in range(51))
    status = api_status("/api/file/upload", fields=fields)
    assert status == "400", f"Expected 400 for too many tags, got {status}"


def verify_oversized_upload_rejected():
    # Test config caps max_upload_size at 8192 bytes; anything larger must
    # 413 via the bodySizeMiddleware MaxBytesReader.
    machine.succeed("head -c 16384 /dev/urandom > /tmp/oversized.bin")
    status = api_status(
        "/api/file/upload",
        fields=[("file", "@/tmp/oversized.bin"), ("plain", "true")],
    )
    assert status == "413", f"Expected 413 for oversized upload, got {status}"


def verify_unauth_rejected():
    status = api_status("/api/account/files", cookies=None)
    assert status == "401", f"Expected 401 on unauth /api/account/files, got {status}"


def verify_malformed_upload_token_rejected():
    # uuid.Parse failure in hasUploadOrSessionTokenMiddleware must 401.
    status = api_status(
        "/api/file/upload",
        cookies=None,
        fields=[
            ("file", f"@{DUMMY_FILE}"),
            ("plain", "true"),
            ("upload_token", "not-a-uuid"),
        ],
    )
    assert status == "401", (
        f"Expected 401 for malformed UUID upload_token, got {status}"
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


def verify_upload_token_nickname_length_boundary():
    ok_token = api_call(
        "/api/account/upload_token", fields={"nickname": "x" * 64}
    )
    assert len(ok_token) == 36, f"Expected UUID for 64-char nickname, got {ok_token!r}"
    api_call(
        "/api/account/upload_token",
        method="DELETE",
        fields={"upload_token": ok_token},
    )

    status = api_status(
        "/api/account/upload_token", fields={"nickname": "x" * 65}
    )
    assert status == "400", f"Expected 400 for 65-char nickname, got {status}"


def verify_private_file_enumeration_safe(private_file):
    # Private & nonexistent files must return identical responses so attackers
    # can't probe filename existence by status code.
    toggle_public(private_file)
    status = api_status(f"/{private_file}", cookies=None)
    assert status == "307", (
        f"Expected 307 for unauth fetch of private file, got {status}"
    )


def verify_bulk_delete():
    api_call("/api/account/files", method="DELETE")
    listing = api_get("/api/account/files")
    assert listing["count"] == 0, (
        f"Expected 0 files after bulk delete, got {listing['count']}"
    )


def get_file_record(file_name):
    listing = api_get("/api/account/files")
    for f in listing["files"]:
        if f["FileName"] == file_name:
            return f
    raise AssertionError(f"File {file_name} not in listing")


def verify_view_counter_bumps():
    # Fresh file 1: plain GET bumps, and same IP doesn't double-count
    f1 = upload_file()
    assert get_file_record(f1)["ViewsCount"] == 0, "Expected 0 views on fresh upload"

    api_call(f"/{f1}", cookies=None)
    assert get_file_record(f1)["ViewsCount"] == 1, "Expected 1 view after GET"

    api_call(f"/{f1}", cookies=None)
    assert get_file_record(f1)["ViewsCount"] == 1, "Expected dedup to keep count at 1"

    # HEAD must not bump
    f2 = upload_file()
    machine.succeed(f"curl -f -I 'http://localhost:{PORT}/{f2}' -o /dev/null")
    assert get_file_record(f2)["ViewsCount"] == 0, "HEAD must not bump views"

    # Range request must not bump
    f3 = upload_file()
    machine.succeed(f"curl -f -r 0-100 'http://localhost:{PORT}/{f3}' -o /dev/null")
    assert get_file_record(f3)["ViewsCount"] == 0, "Range request must not bump views"

    delete_file_entry(f1)
    delete_file_entry(f2)
    delete_file_entry(f3)


def verify_duplicate_tag_rejected():
    f = upload_file()
    add_tag(f, "alpha")
    status = api_status(
        "/api/account/file/tag", fields={"file_name": f, "tag": "alpha"}
    )
    assert status == "400", f"Expected 400 for duplicate tag, got {status}"
    delete_file_entry(f)


def verify_tag_dedup_on_upload():
    f = upload_file(fields=[("tag", "alpha"), ("tag", "alpha"), ("tag", "beta")])
    record = get_file_record(f)
    tags = sorted(t["Name"] for t in record.get("Tags", []) or [])
    assert tags == ["alpha", "beta"], f"Expected deduped tags [alpha, beta], got {tags}"
    delete_file_entry(f)


def verify_future_expiry_happy_path():
    future = int(time.time()) + 86400  # 1 day out
    f = upload_file(fields={"expiry_timestamp": str(future)})
    record = get_file_record(f)
    assert record["ExpiryDate"] != "0001-01-01T00:00:00Z", (
        f"Expected non-zero ExpiryDate, got {record['ExpiryDate']}"
    )
    # File is reachable until it expires
    api_call(f"/{f}", cookies=None)
    delete_file_entry(f)


def verify_expiry_date_form_field():
    f = upload_file(fields={"expiry_date": "2099-12-31"})
    record = get_file_record(f)
    assert record["ExpiryDate"].startswith("2099-12-31"), (
        f"Expected ExpiryDate to start with 2099-12-31, got {record['ExpiryDate']}"
    )
    delete_file_entry(f)


def verify_file_expires():
    now = int(machine.succeed("date +%s").strip())
    expire_at = now + 3
    f = upload_file(fields={"expiry_timestamp": str(expire_at)})

    # Reachable immediately
    api_call(f"/{f}", cookies=None)

    # Wait past the expiry. GetFileByName filters by expiry_date > now, so
    # the file becomes invisible to reads even before the cron sweep runs.
    time.sleep(5)

    status = api_status(f"/{f}", cookies=None)
    assert status == "307", f"Expected 307 for expired file, got {status}"


def verify_pagination():
    # Backend sends 8 files per page. Upload 10 so the
    # second page has something hehe.
    files = [upload_file() for _ in range(10)]

    page1 = api_get("/api/account/files?skip=0")
    assert page1["count"] == 10, f"Expected count=10, got {page1['count']}"
    assert len(page1["files"]) == 8, f"Expected 8 on page1, got {len(page1['files'])}"

    page2 = api_get("/api/account/files?skip=8")
    assert page2["count"] == 10, f"Expected count=10, got {page2['count']}"
    assert len(page2["files"]) == 2, f"Expected 2 on page2, got {len(page2['files'])}"

    page1_names = {f["FileName"] for f in page1["files"]}
    page2_names = {f["FileName"] for f in page2["files"]}
    assert page1_names.isdisjoint(page2_names), "Pages must not overlap"
    assert page1_names | page2_names == set(files), (
        "Union of pages must equal all uploaded files"
    )

    for f in files:
        delete_file_entry(f)


def verify_combined_query_params():
    small = upload_file()
    machine.succeed("head -c 2048 /dev/urandom > /tmp/combo_large.bin")
    large = upload_file(src="/tmp/combo_large.bin")

    add_tag(small, "combo")
    add_tag(large, "combo")
    toggle_public(small)  # small is private; large stays public

    # tag + filter compose: tag=combo & filter=public → only large
    listing = api_get("/api/account/files?tag=combo&filter=public")
    assert listing["count"] == 1, (
        f"Expected 1 file matching tag=combo&filter=public, got {listing['count']}"
    )
    assert listing["files"][0]["FileName"] == large

    # tag + sort compose: tag=combo & sort=file_size&desc=true → [large, small]
    listing = api_get("/api/account/files?tag=combo&sort=file_size&desc=true")
    actual = [f["FileName"] for f in listing["files"]]
    assert actual == [large, small], (
        f"Expected [large, small] for tag=combo sorted by size desc, got {actual}"
    )

    delete_file_entry(small)
    delete_file_entry(large)


def verify_html_pages():
    expected_authed = {
        "/": "200",
        "/login": "307",
        "/register": "307",
        "/gallery": "200",
        "/settings": "200",
        "/tokens": "200",
        "/admin": "200",
    }
    for page, expected in expected_authed.items():
        status = api_status(page)
        assert status == expected, (
            f"Authed GET {page}: expected {expected}, got {status}"
        )

    expected_unauth = {
        "/": "200",
        "/login": "200",
        "/register": "200",
        "/gallery": "307",
        "/settings": "307",
        "/tokens": "307",
        "/admin": "307",
    }
    for page, expected in expected_unauth.items():
        status = api_status(page, cookies=None)
        assert status == expected, (
            f"Unauth GET {page}: expected {expected}, got {status}"
        )


# A is the initial registrant, so admin, ID=1. B will be the next user, ID=2.
ADMIN_ID = 1
USER_B_ID = 2


def verify_admin_self_delete_blocked():
    # Admin endpoint refuses to delete the calling account (different from
    # /api/account/ self-delete which is allowed).
    status = api_status("/api/admin/user", method="DELETE", fields={"id": ADMIN_ID})
    assert status == "400", f"Expected 400 admin self-delete refuse, got {status}"


def verify_invite_code_deletion():
    code = admin_give_invite_code(ADMIN_ID, uses=1)
    api_call(
        "/api/account/invite_code",
        method="DELETE",
        fields={"invite_code": code},
    )
    status = register_status(code)
    assert status == "400", f"Expected 400 for deleted invite, got {status}"


def verify_invite_use_counter():
    code = admin_give_invite_code(ADMIN_ID, uses=2)
    register_with_invite_code(code, cookies="/tmp/cookies_inv_1.txt")
    register_with_invite_code(code, cookies="/tmp/cookies_inv_2.txt")
    status = register_status(code)
    assert status == "400", f"Expected 400 for exhausted invite, got {status}"


def register_user_b():
    code = admin_give_invite_code(ADMIN_ID)
    register_with_invite_code(code, cookies=COOKIES_B)


def verify_cross_user_isolation():
    """User A cannot tamper with user B's files. Returns B's filename for later use."""
    b_file = upload_file(cookies=COOKIES_B)
    toggle_public(b_file, cookies=COOKIES_B)  # mark private

    # A can't delete B's file
    status = api_status(
        "/api/account/file",
        method="DELETE",
        fields={"file_name": b_file},
    )
    assert status == "404", f"Expected 404 cross-user delete, got {status}"

    # A can't tag B's file
    status = api_status(
        "/api/account/file/tag",
        fields={"file_name": b_file, "tag": "x"},
    )
    assert status == "404", f"Expected 404 cross-user tag, got {status}"

    # A can't toggle visibility on B's file
    status = api_status(
        "/api/account/file/public",
        fields={"file_name": b_file},
    )
    assert status == "404", f"Expected 404 cross-user toggle, got {status}"

    # Unauth fetch of B's private file → 307 (matches "not found" — enumeration-safe)
    status = api_status(f"/{b_file}", cookies=None)
    assert status == "307", (
        f"Expected 307 unauth fetch of B's private file, got {status}"
    )

    return b_file


def verify_admin_force_delete_files(b_file):
    admin_delete_files(USER_B_ID)
    listing = api_get("/api/account/files", cookies=COOKIES_B)
    assert listing["count"] == 0, (
        f"Expected B to have 0 files after admin wipe, got {listing['count']}"
    )
    # B's file URL also gone (deleted ⇒ 307)
    status = api_status(f"/{b_file}", cookies=None)
    assert status == "307", f"Expected 307 for admin-deleted file, got {status}"


def verify_admin_revokes_upload_tokens():
    # B creates an upload token, admin revokes all of B's tokens, B can't use it
    b_token = api_call(
        "/api/account/upload_token",
        fields={"nickname": "b-token"},
        cookies=COOKIES_B,
    )
    admin_delete_upload_tokens(USER_B_ID)
    status = api_status(
        "/api/file/upload",
        cookies=None,
        fields=[
            ("file", f"@{DUMMY_FILE}"),
            ("plain", "true"),
            ("upload_token", b_token),
        ],
    )
    assert status == "401", f"Expected 401 for admin-revoked upload token, got {status}"


def verify_admin_revokes_sessions():
    admin_delete_sessions(USER_B_ID)
    status = api_status("/api/account/files", cookies=COOKIES_B)
    assert status == "401", f"Expected 401 for B after admin session wipe, got {status}"


def verify_admin_delete_user():
    admin_delete_account(USER_B_ID)
    # Account is gone — admin operations targeting it now return 404
    status = api_status(
        "/api/admin/give_invite_code",
        fields={"id": USER_B_ID, "uses": 1},
    )
    assert status == "404", f"Expected 404 for admin op on deleted user, got {status}"


def verify_account_self_delete():
    # Self delete throwaway account
    cookies_c = "/tmp/cookies_c.txt"
    code = admin_give_invite_code(ADMIN_ID)
    register_with_invite_code(code, cookies=cookies_c)

    # Upload a file to confirm file deletion cascading
    c_file = upload_file(cookies=cookies_c)

    api_call("/api/account/", method="DELETE", cookies=cookies_c)

    # Session is dead
    status = api_status("/api/account/files", cookies=cookies_c)
    assert status == "401", f"Expected 401 after self-delete, got {status}"

    # File blob/row gone — same 307 as any other missing/private file
    status = api_status(f"/{c_file}", cookies=None)
    assert status == "307", f"Expected 307 for deleted user's file, got {status}"


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
    verify_sorting(small_file, large_file)
    verify_tagging(small_file, large_file)
    verify_public_private_toggle(small_file, large_file)
    verify_tag_removal(small_file, large_file)
    verify_file_deletion(small_file, large_file)

    verify_tag_too_long_rejected()
    verify_past_expiry_rejected()
    verify_malformed_expiry_date_rejected()
    verify_expiry_too_far_rejected()
    verify_too_many_tags_rejected()
    verify_oversized_upload_rejected()
    verify_unauth_rejected()
    verify_malformed_upload_token_rejected()

    private_file = verify_upload_token_lifecycle()
    verify_upload_token_nickname_length_boundary()
    verify_private_file_enumeration_safe(private_file)
    verify_bulk_delete()
    verify_view_counter_bumps()
    verify_duplicate_tag_rejected()
    verify_tag_dedup_on_upload()
    verify_future_expiry_happy_path()
    verify_expiry_date_form_field()
    verify_file_expires()
    verify_pagination()
    verify_combined_query_params()
    verify_html_pages()

    verify_admin_self_delete_blocked()
    verify_invite_code_deletion()
    register_user_b()
    b_file = verify_cross_user_isolation()
    verify_admin_force_delete_files(b_file)
    verify_admin_revokes_upload_tokens()
    verify_admin_revokes_sessions()
    verify_admin_delete_user()
    verify_invite_use_counter()
    verify_account_self_delete()

    verify_logout_invalidates_session()


main()
