const CHUNK_SIZE = 2 * 1024 * 1024; // 2MB per slice
const MAX_IMAGE_DIMENSION = 2048;
const JPEG_QUALITY = 0.85;
const COMPRESS_THRESHOLD = 2 * 1024 * 1024; // only compress images > 2MB

export const COMPRESSIBLE_IMAGE_TYPES = new Set(["image/jpeg", "image/heic", "image/heif", "image/png", "image/bmp", "image/tiff"]);

// Resize and re-encode large images to reduce memory pressure on mobile browsers.
// Returns the original file unchanged if it is small or an unsupported type.
export async function compressImageIfNeeded(file: File): Promise<File> {
  if (!COMPRESSIBLE_IMAGE_TYPES.has(file.type) || file.size <= COMPRESS_THRESHOLD) {
    return file;
  }

  const bitmap = await createImageBitmap(file);
  const { width, height } = bitmap;
  const ratio = Math.min(MAX_IMAGE_DIMENSION / width, MAX_IMAGE_DIMENSION / height, 1);
  const targetW = Math.round(width * ratio);
  const targetH = Math.round(height * ratio);

  const canvas = document.createElement("canvas");
  canvas.width = targetW;
  canvas.height = targetH;
  const ctx = canvas.getContext("2d")!;
  ctx.fillStyle = "#ffffff";
  ctx.fillRect(0, 0, targetW, targetH);
  ctx.drawImage(bitmap, 0, 0, targetW, targetH);
  bitmap.close();

  const blob = await new Promise<Blob>((resolve, reject) => {
    canvas.toBlob((b) => (b ? resolve(b) : reject(new Error("canvas.toBlob returned null"))), "image/jpeg", JPEG_QUALITY);
  });

  const stem = file.name.replace(/\.[^.]+$/, "");
  return new File([blob], `${stem}.jpg`, { type: "image/jpeg" });
}

// Read a file in 2MB slices to avoid triggering NotReadableError on iOS,
// where a single large arrayBuffer() call can invalidate the File reference.
export async function readFileInChunks(file: File | Blob): Promise<Uint8Array> {
  const result = new Uint8Array(file.size);
  let offset = 0;
  while (offset < file.size) {
    const chunk = file.slice(offset, Math.min(offset + CHUNK_SIZE, file.size));
    const buf = await chunk.arrayBuffer();
    result.set(new Uint8Array(buf), offset);
    offset += buf.byteLength;
  }
  return result;
}
