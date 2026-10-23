export interface UploadToken {
  id: number;
  created_at: string;
  name: string;
  last_used: string | null;
}

export interface CreatedUploadToken extends UploadToken {
  token: string;
}

export interface InviteCode {
  code: string;
  uses: number;
  expiry_date: string;
}

export interface TokensData {
  upload_tokens: UploadToken[];
  invite_codes: InviteCode[];
}

async function request(url: string, method = 'GET', fields?: Record<string, string>): Promise<Response> {
  const body = fields ? new FormData() : undefined;
  for (const [key, value] of Object.entries(fields ?? {})) body!.append(key, value);
  const response = await fetch(url, { method, body, cache: 'no-store' });
  if (!response.ok) {
    const text = await response.text();
    throw new Error(response.headers.get('content-type')?.startsWith('text/plain') && text
      ? text.slice(0, 200)
      : 'The request failed. Please try again.');
  }
  return response;
}

export async function fetchTokens(): Promise<TokensData> {
  const [tokensResponse, invitesResponse] = await Promise.all([
    request('/api/account/upload_tokens'), request('/api/account/invite_code'),
  ]);
  return { upload_tokens: await tokensResponse.json(), invite_codes: await invitesResponse.json() };
}

export async function createUploadToken(name: string): Promise<CreatedUploadToken> {
  const response = await request('/api/account/upload_tokens', 'POST', { nickname: name });
  return response.json();
}

export async function deleteUploadToken(id: number): Promise<void> {
  await request('/api/account/upload_tokens', 'DELETE', { id: String(id) });
}

export async function deleteInviteCode(code: string): Promise<void> {
  await request('/api/account/invite_code', 'DELETE', { invite_code: code });
}
