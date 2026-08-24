import { act, renderHook } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import type React from 'react';
import { useFileUpload } from './useFileUpload';

const createClipboardItem = (type: string, file?: File): DataTransferItem => ({
  kind: type.startsWith('text/') ? 'string' : 'file',
  type,
  getAsFile: () => file ?? null,
  getAsString: vi.fn(),
  webkitGetAsEntry: () => null,
} as unknown as DataTransferItem);

const createPasteEvent = (items: DataTransferItem[]) => ({
  clipboardData: {
    items,
    types: items.map((item) => item.type),
  },
  preventDefault: vi.fn(),
}) as unknown as React.ClipboardEvent<HTMLTextAreaElement>;

describe('useFileUpload', () => {
  it('leaves text tables to the textarea when the clipboard also contains an image', async () => {
    const clipboardImage = new File(['rendered table'], 'table.png', { type: 'image/png' });
    const event = createPasteEvent([
      createClipboardItem('text/plain'),
      createClipboardItem('image/png', clipboardImage),
    ]);
    const { result } = renderHook(() => useFileUpload());

    await act(async () => {
      await result.current.handlePaste(event);
    });

    expect(event.preventDefault).not.toHaveBeenCalled();
    expect(result.current.selectedImages).toHaveLength(0);
  });

  it('adds a pasted image when no text representation is available', async () => {
    const clipboardImage = new File(['image bytes'], 'clipboard.png', { type: 'image/png' });
    const event = createPasteEvent([createClipboardItem('image/png', clipboardImage)]);
    const { result } = renderHook(() => useFileUpload());

    await act(async () => {
      await result.current.handlePaste(event);
    });

    expect(event.preventDefault).toHaveBeenCalledTimes(1);
    expect(result.current.selectedImages).toHaveLength(1);
    expect(result.current.selectedImages[0]).toMatchObject({
      name: 'clipboard.png',
      mimeType: 'image/png',
      isPdf: false,
      isAudio: false,
    });
    expect(result.current.selectedImages[0].dataUrl).toMatch(/^data:image\/png;base64,/);
  });
});
