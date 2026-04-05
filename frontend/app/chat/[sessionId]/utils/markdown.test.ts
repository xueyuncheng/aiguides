import { describe, expect, it } from 'vitest';
import { preprocessMarkdown } from './markdown';

describe('preprocessMarkdown', () => {
  it('wraps raw svg blocks as fenced svg code blocks', () => {
    const content = '图如下：\n<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10"><rect width="10" height="10" /></svg>';

    expect(preprocessMarkdown(content)).toContain('```svg\n<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10"><rect width="10" height="10" /></svg>\n```');
  });

  it('does not wrap svg that is already in a fenced code block', () => {
    const content = '```svg\n<svg viewBox="0 0 10 10"></svg>\n```';

    expect(preprocessMarkdown(content)).toBe(content);
  });

  it('escapes currency outside code fences without touching svg code', () => {
    const content = '$100\n<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10"><text>$100</text></svg>';

    expect(preprocessMarkdown(content)).toBe('\\$100\n\n```svg\n<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10"><text>$100</text></svg>\n```\n');
  });
});
