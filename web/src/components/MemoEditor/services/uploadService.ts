import { create } from "@bufbuild/protobuf";
import { attachmentServiceClient } from "@/connect";
import type { Attachment } from "@/types/proto/api/v1/attachment_service_pb";
import { AttachmentSchema, MotionMediaSchema } from "@/types/proto/api/v1/attachment_service_pb";
import { COMPRESSIBLE_IMAGE_TYPES, compressImageIfNeeded, readFileInChunks } from "@/utils/fileUtils";
import type { LocalFile } from "../types/attachment";

export const uploadService = {
  async uploadFiles(localFiles: LocalFile[]): Promise<Attachment[]> {
    if (localFiles.length === 0) return [];

    const attachments: Attachment[] = [];

    for (const localFile of localFiles) {
      const { file, motionMedia } = localFile;

      const uploadFile = COMPRESSIBLE_IMAGE_TYPES.has(file.type) ? await compressImageIfNeeded(file) : file;
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
