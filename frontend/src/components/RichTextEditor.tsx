import React, { lazy, Suspense } from 'react';
import type { RichTextEditorProps } from './RichTextEditorImpl';

/**
 * Lazy boundary around the Jodit-based editor.
 *
 * Jodit is roughly a megabyte of JavaScript. It was reachable from Dashboard,
 * which is on the first-paint path, so every user downloaded the whole editor
 * before seeing their own dashboard — including the majority who never open an
 * editing surface at all.
 *
 * The boundary lives here rather than at each call site so all six consumers get
 * the benefit without changing, and so nobody has to remember to wrap it.
 */
const RichTextEditorImpl = lazy(() => import('./RichTextEditorImpl'));

/** Placeholder sized like the editor, so the layout does not jump when it loads. */
const EditorSkeleton: React.FC<{ height?: number }> = ({ height = 400 }) => (
  <div
    className="w-full animate-pulse rounded-md border border-border bg-muted/40"
    style={{ height }}
    role="status"
    aria-live="polite"
  >
    <span className="sr-only">Loading editor</span>
  </div>
);

export const RichTextEditor: React.FC<RichTextEditorProps> = (props) => (
  <Suspense fallback={<EditorSkeleton height={props.height} />}>
    <RichTextEditorImpl {...props} />
  </Suspense>
);

export type { RichTextEditorProps };
