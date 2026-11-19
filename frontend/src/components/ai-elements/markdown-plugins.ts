// 职责: 给 Streamdown 的 markdown 渲染管线提供统一的 rehype 插件数组，允许内联 base64 图片。
// 关联文件:
//   frontend/src/components/ai-elements/response.tsx
//   frontend/src/components/ai-elements/message.tsx
//   frontend/src/components/ai-elements/reasoning.tsx
// 遵循的 Rules:
//   .agents/rules/frontend-general.md
// 注意事项:
//   defaultRehypePlugins 的键随 streamdown 版本变化（2.7.1 只有 raw/sanitize/harden，没有 katex）。
//   往 unified 的插件数组里放 undefined 会抛 “Expected usable value, not `undefined`”，
//   整个 markdown 渲染随之崩溃到 500 错误页，所以这里只保留实际存在的插件。
//   sanitize 的 schema 基于 streamdown 默认值做浅合并：整体替换 protocols 会丢掉
//   默认的 tel/streamdown 协议和 code.metastring，导致链接与代码块元信息被清洗掉。
import { defaultRehypePlugins, Streamdown } from 'streamdown';
import type { ComponentProps } from 'react';

type RehypePlugins = NonNullable<ComponentProps<typeof Streamdown>['rehypePlugins']>;

const sanitizePlugin = defaultRehypePlugins.sanitize as [any, Record<string, any>];
const defaultSanitizeSchema = sanitizePlugin[1];

// Streamdown runs rehype-sanitize before rehype-harden, and the default schema only
// allows http/https on src, so base64 data URLs are stripped before harden can allow
// them. Providers such as Gemini return generated images inline as markdown data URLs.
const DATA_IMAGE_SANITIZE_SCHEMA = {
  ...defaultSanitizeSchema,
  protocols: {
    ...defaultSanitizeSchema.protocols,
    src: [...(defaultSanitizeSchema.protocols?.src ?? []), 'data'],
  },
};

export const markdownRehypePlugins = [
  defaultRehypePlugins.raw,
  [sanitizePlugin[0], DATA_IMAGE_SANITIZE_SCHEMA],
  defaultRehypePlugins.harden,
].filter(Boolean) as RehypePlugins;
