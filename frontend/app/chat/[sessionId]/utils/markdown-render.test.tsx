import { render } from '@testing-library/react';
import ReactMarkdown from 'react-markdown';
import { describe, expect, it } from 'vitest';
import { markdownComponents, markdownRehypePlugins, markdownRemarkPlugins, preprocessMarkdown } from './markdown';

describe('markdown SVG rendering', () => {
  it('renders svg preview from a full assistant response', () => {
    const svgBlock = [
      '```svg',
      '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 500 200">',
      '  <title>Next.js to Go Backend Architecture</title>',
      '  <style>',
      '    .box { rx: 8; ry: 8; fill: #ef4444; }',
      '    .text { fill: white; font-family: sans-serif; font-size: 14px; font-weight: bold; }',
      '    .label { fill: #333; font-family: sans-serif; font-size: 12px; }',
      '  </style>',
      '  <rect x="50" y="50" width="120" height="80" class="box" />',
      '  <text x="110" y="95" text-anchor="middle" class="text">Next.js</text>',
      '  <text x="110" y="150" text-anchor="middle" class="label">Frontend</text>',
      '  <line x1="170" y1="90" x2="330" y2="90" stroke="#ef4444" stroke-width="2" stroke-dasharray="4" />',
      '  <text x="250" y="80" text-anchor="middle" class="label">API Request</text>',
      '  <rect x="330" y="50" width="120" height="80" class="box" />',
      '  <text x="390" y="95" text-anchor="middle" class="text">Go</text>',
      '  <text x="390" y="150" text-anchor="middle" class="label">Backend</text>',
      '</svg>',
      '```',
    ].join('\n');

    const fullContent = [
      '没问题！考虑到你正在进行 Next.js 与 Go 的全栈开发，我为你设计了一个 SVG 模块化结构的示例，它使用了你喜欢的红色主题，展示了前后端交互的抽象概念。',
      '',
      svgBlock,
      '',
      '这个 SVG 代码完全符合你的需求：',
    ].join('\n');

    const { container } = render(
      <ReactMarkdown
        remarkPlugins={markdownRemarkPlugins}
        rehypePlugins={markdownRehypePlugins}
        components={markdownComponents}
      >
        {preprocessMarkdown(fullContent)}
      </ReactMarkdown>
    );

    const svg = container.querySelector('svg title')?.closest('svg');

    expect(svg).not.toBeNull();
    expect(svg?.getAttribute('width')).toBe('500');
    expect(svg?.getAttribute('height')).toBe('200');
    expect(container.innerHTML).not.toContain('<pre><div');
    expect(container.textContent).toContain('这个 SVG 代码完全符合你的需求');
  });
});
