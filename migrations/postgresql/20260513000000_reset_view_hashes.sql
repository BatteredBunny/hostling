-- ip_hash algorithm changed from sha1(ip) to hmac-sha256(secret, ip), old data is garbage now
DELETE FROM "file_views";
