#!/usr/bin/env python3
import os
import shutil
import subprocess
import time

from selenium import webdriver
from selenium.webdriver.chrome.options import Options
from selenium.webdriver.chrome.service import Service
from selenium.webdriver.common.by import By
from selenium.webdriver.support import expected_conditions as EC
from selenium.webdriver.support.ui import WebDriverWait

PORT = os.environ["PORT"]
TOKEN = os.environ["TOKEN"]
MASCOT = os.environ.get("MASCOT", "")
BASE = f"http://localhost:{PORT}"
COOKIES = "/tmp/cookies.txt"


def curl(*args: str) -> str:
    result = subprocess.run(
        ["curl", "-f", *args],
        capture_output=True, text=True, check=True,
    )
    return result.stdout


# Register account
curl("-c", COOKIES, "-L", "-X", "POST", "--data-urlencode", f"code={TOKEN}", f"{BASE}/api/auth/register")
print("Registered")

# Upload the example images
for path, name in [(MASCOT, "mascot.png"), (MASCOT, "mascot.png")]:
    if not path:
        continue
    curl("-b", COOKIES, "-F", f"file=@{path}", f"{BASE}/api/file/upload")
    print(f"Uploaded {name}")

auth = ""
with open(COOKIES) as f:
    for line in f:
        parts = line.strip().split("\t")
        if len(parts) >= 7 and parts[5] == "auth":
            auth = parts[6]
            break

if not auth:
    raise RuntimeError("Could not find auth cookie in cookie jar")

chromium_bin = shutil.which("chromium") or shutil.which("chromium-browser")
chromedriver_bin = shutil.which("chromedriver")

options = Options()
if chromium_bin:
    options.binary_location = chromium_bin
options.add_argument("--headless=new")
options.add_argument("--no-sandbox")
options.add_argument("--disable-gpu")
options.add_argument("--disable-dev-shm-usage")
options.add_argument("--disable-software-rasterizer")
options.add_argument("--window-size=1280,900")

service = Service(executable_path=chromedriver_bin) if chromedriver_bin else Service()
driver = webdriver.Chrome(service=service, options=options)

try:
    driver.get(BASE)
    driver.add_cookie({"name": "auth", "value": auth})

    # Enable dark mode
    driver.execute_cdp_cmd("Emulation.setEmulatedMedia", {
        "features": [{"name": "prefers-color-scheme", "value": "dark"}]
    })

    # Index
    driver.get(f"{BASE}/")
    time.sleep(1)
    driver.save_screenshot("/tmp/upload.png")
    print("Screenshot: /tmp/upload.png")

    # Gallery
    driver.get(f"{BASE}/gallery")
    WebDriverWait(driver, 60).until(
        EC.presence_of_element_located((By.CSS_SELECTOR, ".file-entry"))
    )
    time.sleep(0.5)
    driver.save_screenshot("/tmp/gallery.png")
    print("Screenshot: /tmp/gallery.png")

    # Gallery modal
    driver.find_element(By.CSS_SELECTOR, ".file-preview").click()
    WebDriverWait(driver, 10).until(
        EC.visibility_of_element_located((By.CSS_SELECTOR, ".file-modal-visible"))
    )
    time.sleep(0.5)
    driver.save_screenshot("/tmp/modal.png")
    print("Screenshot: /tmp/modal.png")

    # Admin page — scroll to bottom to show full content
    driver.get(f"{BASE}/admin")
    time.sleep(1)
    driver.execute_script("window.scrollTo(0, document.body.scrollHeight);")
    time.sleep(0.5)
    driver.save_screenshot("/tmp/admin.png")
    print("Screenshot: /tmp/admin.png")
finally:
    driver.quit()

print("Done.")
