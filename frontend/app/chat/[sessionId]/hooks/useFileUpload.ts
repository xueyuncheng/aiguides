import { useCallback, useState } from 'react';
import type { SelectedImage } from '../types';
import {
  MAX_IMAGE_COUNT,
  MAX_IMAGE_SIZE_BYTES,
  MAX_PDF_SIZE_BYTES,
  MAX_AUDIO_SIZE_BYTES,
  IMAGE_COUNT_ERROR,
  IMAGE_SIZE_ERROR,
  PDF_SIZE_ERROR,
  AUDIO_SIZE_ERROR,
  IMAGE_TYPE_ERROR,
  IMAGE_READ_ERROR,
} from '../constants';

const AUDIO_MIME_BY_EXTENSION: Record<string, string> = {
  '.mp3': 'audio/mpeg',
  '.m4a': 'audio/x-m4a',
  '.mp4': 'audio/mp4',
  '.wav': 'audio/wav',
  '.aac': 'audio/aac',
  '.webm': 'audio/webm',
  '.ogg': 'audio/ogg',
};

const getFileExtension = (fileName: string) => {
  const lastDot = fileName.lastIndexOf('.');
  if (lastDot < 0) {
    return '';
  }

  return fileName.slice(lastDot).toLowerCase();
};

const inferMimeType = (file: File) => {
  if (file.type) {
    return file.type;
  }

  const extension = getFileExtension(file.name);
  if (extension === '.pdf') {
    return 'application/pdf';
  }

  return AUDIO_MIME_BY_EXTENSION[extension] || '';
};

const normalizeDataUrlMimeType = (dataUrl: string, mimeType: string) => {
  if (!dataUrl.startsWith('data:') || !mimeType) {
    return dataUrl;
  }

  const commaIndex = dataUrl.indexOf(',');
  if (commaIndex < 0) {
    return dataUrl;
  }

  const header = dataUrl.slice(0, commaIndex);
  const payload = dataUrl.slice(commaIndex + 1);
  const encodedSuffix = header.includes(';base64') ? ';base64' : '';
  return `data:${mimeType}${encodedSuffix},${payload}`;
};

const readFileAsDataUrl = (file: File) => new Promise<string>((resolve, reject) => {
  const reader = new FileReader();
  reader.onload = () => resolve(reader.result as string);
  reader.onerror = () => reject(reader.error);
  reader.readAsDataURL(file);
});

const createImageId = () => {
  if (typeof crypto !== 'undefined') {
    if ('randomUUID' in crypto) {
      return (crypto as Crypto).randomUUID();
    }
    if ('getRandomValues' in crypto) {
      const bytes = new Uint8Array(16);
      (crypto as Crypto).getRandomValues(bytes);
      const hex = Array.from(bytes)
        .map((value) => value.toString(16).padStart(2, '0'))
        .join('');
      return `img-${hex}`;
    }
  }
  return `img-${Date.now()}-${Math.random().toString(36).slice(2)}`;
};

export function useFileUpload() {
  const [selectedImages, setSelectedImages] = useState<SelectedImage[]>([]);
  const [imageError, setImageError] = useState<string | null>(null);

  const addImagesFromFiles = useCallback(async (files: File[]) => {
    if (files.length === 0) return;

    setImageError(null);
    const remainingSlots = MAX_IMAGE_COUNT - selectedImages.length;
    if (remainingSlots <= 0) {
      setImageError(IMAGE_COUNT_ERROR);
      return;
    }

    const nextImages: SelectedImage[] = [];
    let errorMessage: string | null = null;
    const limitedFiles = files.slice(0, remainingSlots);

    for (const [index, file] of limitedFiles.entries()) {
      const mimeType = inferMimeType(file);
      const isPdf = mimeType === 'application/pdf';
      const isImage = mimeType.startsWith('image/');
      const isAudio = mimeType.startsWith('audio/');
      if (!isImage && !isPdf && !isAudio) {
        if (!errorMessage) {
          errorMessage = IMAGE_TYPE_ERROR;
        }
        continue;
      }
      if (isPdf && file.size > MAX_PDF_SIZE_BYTES) {
        if (!errorMessage) {
          errorMessage = PDF_SIZE_ERROR;
        }
        continue;
      }
      if (isImage && file.size > MAX_IMAGE_SIZE_BYTES) {
        if (!errorMessage) {
          errorMessage = IMAGE_SIZE_ERROR;
        }
        continue;
      }
      if (isAudio && file.size > MAX_AUDIO_SIZE_BYTES) {
        if (!errorMessage) {
          errorMessage = AUDIO_SIZE_ERROR;
        }
        continue;
      }

      try {
        const rawDataUrl = await readFileAsDataUrl(file);
        const dataUrl = normalizeDataUrlMimeType(rawDataUrl, mimeType);
        const imageId = createImageId();
        const fallbackName = isPdf
          ? `file-${index + 1}.pdf`
          : isAudio
            ? `audio-${index + 1}`
            : `clipboard-image-${index + 1}`;
        nextImages.push({
          id: imageId,
          dataUrl,
          name: file.name || fallbackName,
          mimeType,
          isPdf,
          isAudio,
        });
      } catch (error) {
        console.error('Error reading file:', error);
        if (!errorMessage) {
          errorMessage = IMAGE_READ_ERROR;
        }
      }
    }

    if (files.length > remainingSlots) {
      if (!errorMessage) {
        errorMessage = IMAGE_COUNT_ERROR;
      }
    }

    if (nextImages.length > 0) {
      setSelectedImages((prev) => [...prev, ...nextImages]);
    }
    if (errorMessage) {
      setImageError(errorMessage);
    }
  }, [selectedImages.length]);

  const handleImageSelect = useCallback(async (event: React.ChangeEvent<HTMLInputElement>) => {
    const files = Array.from(event.target.files ?? []);
    await addImagesFromFiles(files);
    event.target.value = '';
  }, [addImagesFromFiles]);

  const handlePaste = useCallback(async (event: React.ClipboardEvent<HTMLTextAreaElement>) => {
    const items = event.clipboardData?.items;
    if (!items || items.length === 0) return;

    const files: File[] = [];
    for (const item of Array.from(items)) {
      if (item.type.startsWith('image/')) {
        const file = item.getAsFile();
        if (file) {
          files.push(file);
        }
      }
    }

    if (files.length > 0) {
      event.preventDefault();
      await addImagesFromFiles(files);
    }
  }, [addImagesFromFiles]);

  const handleRemoveImage = useCallback((imageId: string) => {
    setSelectedImages((prev) => prev.filter((image) => image.id !== imageId));
  }, []);

  const clearImages = useCallback(() => {
    setSelectedImages([]);
    setImageError(null);
  }, []);

  return {
    selectedImages,
    imageError,
    handleImageSelect,
    handlePaste,
    handleRemoveImage,
    clearImages,
    setImageError,
  };
}
