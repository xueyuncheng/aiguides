import { describe, expect, it } from 'vitest';
import { preprocessMarkdown } from './markdown';

describe('preprocessMarkdown', () => {
  it('leaves raw svg markup unchanged', () => {
    const content = '图如下：\n<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10"><rect width="10" height="10" /></svg>';

    expect(preprocessMarkdown(content)).toBe(content);
  });

  it('preserves fenced svg code as-is', () => {
    const content = '```svg\n<svg viewBox="0 0 10 10"></svg>\n```';

    expect(preprocessMarkdown(content)).toBe(content);
  });

  it('escapes currency outside code fences', () => {
    const content = '$100\n普通文本';

    expect(preprocessMarkdown(content)).toBe('\\$100\n普通文本');
  });
});
