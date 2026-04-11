/**
 * Content type detection for RSS article content
 * Detection order: HTML → Markdown → Plain Text
 */

/**
 * @param {string} content - The article content
 * @returns {'html' | 'markdown' | 'plain'} - Detected content type
 */
export function detectContentType(content) {
  if (!content || typeof content !== 'string') {
    return 'plain';
  }

  const trimmed = content.trim();

  // Empty content
  if (trimmed.length === 0) {
    return 'plain';
  }

  // 1. Check for HTML tags
  const htmlTagRegex = /<\/?[a-z][a-z0-9]*(?:\s+[^>]*)?>/i;
  if (htmlTagRegex.test(trimmed)) {
    return 'html';
  }

  // 2. Check for Markdown patterns
  const markdownPatterns = [
    /^#{1,6}\s+/m,                    // Headings (# to ######)
    /\*\*[^*]+\*\*/,                   // Bold **text**
    /\*(?!\s)[^*]+\*(?!\s)/,          // Italic *text* (not surrounded by spaces)
    /__[^_]+__/,                      // Bold __text__
    /_[^_]+_/,                        // Italic _text_
    /\[[^\]]+\]\([^\)]+\)/,           // Links [text](url)
    /```[\s\S]*?```/,                 // Code blocks
    /`[^`]+`/,                        // Inline code
    /^[\s]*[-*+]\s+/m,               // Unordered lists
    /^[\s]*\d+\.\s+/m,               // Ordered lists
    /^>\s+/m,                        // Blockquotes
    /---+/,                           // Horizontal rules
  ];

  for (const pattern of markdownPatterns) {
    if (pattern.test(trimmed)) {
      return 'markdown';
    }
  }

  // 3. Default to plain text
  return 'plain';
}
