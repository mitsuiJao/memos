import { create } from "@bufbuild/protobuf";
import { attachmentServiceClient } from "@/connect";
import type { Attachment } from "@/types/proto/api/v1/attachment_service_pb";
import { AttachmentSchema, MotionMediaSchema } from "@/types/proto/api/v1/attachment_service_pb";
import { COMPRESSIBLE_IMAGE_TYPES, compressImageIfNeeded, readFileInChunks } from "@/utils/fileUtils";
import type { LocalFile } from "../types/attachment";

// Must match MaxUploadBufferSizeBytes on the server (attachment_service.go).
const MAX_UPLOAD_SIZE_BYTES = 32 * 1024 * 1024;

export const uploadService = {
  async uploadFiles(localFiles: LocalFile[]): Promise<Attachment[]> {
    if (localFiles.length === 0) return [];

    const attachments: Attachment[] = [];

    for (const localFile of localFiles) {
      const { file, motionMedia } = localFile;

      const uploadFile = COMPRESSIBLE_IMAGE_TYPES.has(file.type) ? await compressImageIfNeeded(file) : file;

      if (uploadFile.size > MAX_UPLOAD_SIZE_BYTES) {
        throw new Error(`File "${file.name}" exceeds the maximum upload size of 32 MB.`);
      }

      const buffer = await readFileInChunks(uploadFile);

      const attachment = await attachmentServiceClient.createAttachment({
        attachment: create(AttachmentSchema, {
          filename: uploadFile.name,
          size: BigInt(uploadFile.size),
          type: uploadFile.type,
          content: buffer,
          motionMedia: motionMedia ? create(MotionMediaSchema, motionMedia) : undefined,
        }),
      });
      attachments.push(attachment);
    }

    return attachments;
  },
};
