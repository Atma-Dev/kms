<script setup lang="ts">
/**
 * Wraps whatever a reader is looking at with Modul 4's client-side half: the
 * tiled watermark (FR-1) and the basic copy deterrents (FR-3).
 *
 * What this can and cannot do, plainly: the overlay lives in the reader's own
 * browser, so anyone who opens dev tools can delete it in seconds, and the
 * same goes for the select and context-menu blocking. They raise the cost of a
 * careless copy. They are not evidence, and nothing in the UI should imply
 * they are.
 *
 * The protection that actually holds is server-side, and it is what `strict`
 * switches to: page images with the watermark already in the pixels and no
 * text sent at all. This component is then only drawing the images.
 */
const props = withDefaults(
  defineProps<{
    /** The identity every copy carries: who is reading, and when. */
    text: string
    /** Blocks selection, the context menu and drag when true. */
    protect?: boolean
  }>(),
  { protect: false },
)

/**
 * The tile is one SVG data URL repeated by the browser, rather than a few
 * hundred absolutely positioned spans. One background-image costs one element
 * and reflows with nothing.
 */
const tileSize = 300

/**
 * A small per-reader offset and angle, so two people's overlays do not line up
 * pixel for pixel (FR-1's "posisi bisa sedikit bervariasi per user"). Derived
 * from the watermark text, which already identifies them, so it needs no
 * second source — and being deterministic means the same reader sees a steady
 * pattern rather than one that jumps on every render.
 */
const variation = computed(() => {
  let hash = 0
  for (const char of props.text) hash = (hash * 31 + char.charCodeAt(0)) % 9973
  return {
    offset: hash % tileSize,
    angle: -(26 + (hash % 14)),
  }
})

const tile = computed(() => {
  const { angle } = variation.value
  const half = tileSize / 2
  // The text is XML-escaped before it goes anywhere near the SVG: it is an
  // email address today, but it reaches markup from here either way.
  const escaped = props.text
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')

  const svg =
    `<svg xmlns="http://www.w3.org/2000/svg" width="${tileSize}" height="${tileSize}">` +
    `<text x="${half}" y="${half}" fill="currentColor" font-family="sans-serif" font-size="13" ` +
    `text-anchor="middle" transform="rotate(${angle} ${half} ${half})">${escaped}</text>` +
    `</svg>`

  return `url("data:image/svg+xml;utf8,${encodeURIComponent(svg)}")`
})
</script>

<template>
  <div
    class="relative"
    :class="protect ? 'select-none' : ''"
    @contextmenu="protect ? $event.preventDefault() : undefined"
    @dragstart="protect ? $event.preventDefault() : undefined"
    @copy="protect ? $event.preventDefault() : undefined"
  >
    <slot />

    <!-- pointer-events-none so the overlay never intercepts a click, a link or
         a scroll: a watermark that breaks the page is one someone will ask to
         have turned off. aria-hidden because a screen reader repeating an
         email address every few words is unusable. -->
    <div
      v-if="text"
      class="pointer-events-none absolute inset-0 text-foreground/[0.13] dark:text-foreground/[0.10]"
      aria-hidden="true"
      :style="{
        backgroundImage: tile,
        backgroundPosition: `${variation.offset}px ${variation.offset}px`,
      }"
    />
  </div>
</template>
