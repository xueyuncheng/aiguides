import { useState } from 'react';
import type { SelectedImage } from '../types';
import {
  MAX_IMAGE_COUNT,
  MAX_IMAGE_SIZE_BYTES,
  MAX_PDF_SIZE_BYTES,
  IMAGE_COUNT_ERROR,
  IMAGE_SIZE_ERROR,
  PDF_SIZE_ERROR,
  IMAGE_TYPE_ERROR,
  IMAGE_READ_ERROR,
} from '../constants';

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

  const addImagesFromFiles = async (files: File[]) => {
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
      const isPdf = file.type === 'application/pdf';
      const isImage = file.type.startsWith('image/');
      if (!isImage && !isPdf) {
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

      try {
        const dataUrl = await readFileAsDataUrl(file);
        const imageId = createImageId();
        const fallbackName = isPdf ? `file-${index + 1}.pdf` : `clipboard-image-${index + 1}`;
        nextImages.push({
          id: imageId,
          dataUrl,
          name: file.name || fallbackName,
          isPdf,
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
  };

  const handleImageSelect = async (event: React.ChangeEvent<HTMLInputElement>) => {
    const files = Array.from(event.target.files ?? []);
    await addImagesFromFiles(files);
    event.target.value = '';
  };

  const handlePaste = async (event: React.ClipboardEvent<HTMLTextAreaElement>) => {
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
  };

  const handleRemoveImage = (imageId: string) => {
    setSelectedImages((prev) => prev.filter((image) => image.id !== imageId));
  };

  const clearImages = () => {
    setSelectedImages([]);
    setImageError(null);
  };

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
