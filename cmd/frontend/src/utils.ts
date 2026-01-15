export function formatTimeDate(date: string): string {
  if (!date) return '';
  const d = new Date(date);
  const year = d.getFullYear();
  const month = String(d.getMonth() + 1).padStart(2, '0');
  const day = String(d.getDate()).padStart(2, '0');
  const hours = String(d.getHours()).padStart(2, '0');
  const minutes = String(d.getMinutes()).padStart(2, '0');
  const seconds = String(d.getSeconds()).padStart(2, '0');
  return `${year}-${month}-${day} ${hours}:${minutes}:${seconds}`;
}

export function relativeTime(date: string): string {
  if (!date) return '';
  const d = new Date(date);
  const now = new Date();
  const diff = d.getTime() - now.getTime();
  const absDiff = Math.abs(diff);

  const seconds = Math.floor(absDiff / 1000);
  const minutes = Math.floor(seconds / 60);
  const hours = Math.floor(minutes / 60);
  const days = Math.floor(hours / 24);
  const weeks = Math.floor(days / 7);
  const months = Math.floor(days / 30);
  const years = Math.floor(days / 365);

  let result: string;
  if (years > 0) {
    result = `${years} year${years > 1 ? 's' : ''}`;
  } else if (months > 0) {
    result = `${months} month${months > 1 ? 's' : ''}`;
  } else if (weeks > 0) {
    result = `${weeks} week${weeks > 1 ? 's' : ''}`;
  } else if (days > 0) {
    result = `${days} day${days > 1 ? 's' : ''}`;
  } else if (hours > 0) {
    result = `${hours} hour${hours > 1 ? 's' : ''}`;
  } else if (minutes > 0) {
    result = `${minutes} minute${minutes > 1 ? 's' : ''}`;
  } else {
    result = `${seconds} second${seconds > 1 ? 's' : ''}`;
  }

  if (diff > 0) {
    return `in ${result}`;
  } else {
    return `${result} ago`;
  }
}

export function humanizeBytes(bytes: number): string {
  if (bytes === 0) return '0 B';

  const k = 1000;
  const sizes = ['B', 'kB', 'MB', 'GB', 'TB', 'PB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));

  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

export function mimeIsImage(mimeType: string): boolean {
  return mimeType?.startsWith('image/') ?? false;
}

export function mimeIsVideo(mimeType: string): boolean {
  return mimeType?.startsWith('video/') ?? false;
}

export function mimeIsAudio(mimeType: string): boolean {
  return mimeType?.startsWith('audio/') ?? false;
}

export function hashStringToHSL(str: string): { h: number; s: number; l: number } {
  let hash = 0;
  for (let i = 0; i < str.length; i++) {
    hash = str.charCodeAt(i) + ((hash << 5) - hash);
    hash = hash & hash;
  }

  const h = Math.abs(hash) % 360;
  const s = 65 + (Math.abs(hash >> 8) % 20);
  const l = 45 + (Math.abs(hash >> 16) % 15);

  return { h, s, l };
}

export function hasExpiry(expiryDate: string): boolean {
  return expiryDate !== '0001-01-01T00:00:00Z';
}
