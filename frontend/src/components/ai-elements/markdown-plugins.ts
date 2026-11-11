import { defaultRehypePlugins } from 'streamdown';

const DATA_IMAGE_SANITIZE_SCHEMA = {
  protocols: {
    cite: ['http', 'https'],
    href: ['http', 'https', 'irc', 'ircs', 'mailto', 'xmpp'],
    longDesc: ['http', 'https'],
    src: ['http', 'https', 'data'],
  },
};

const sanitizePlugin = defaultRehypePlugins.sanitize as [any, any];

export const markdownRehypePlugins = [
  defaultRehypePlugins.raw,
  defaultRehypePlugins.katex,
  [sanitizePlugin[0], DATA_IMAGE_SANITIZE_SCHEMA],
  defaultRehypePlugins.harden,
];
