// Renders a schema.org JSON-LD block. Data is our own (never user input),
// but "<" is still escaped so a description can't break out of the script.
export default function JsonLd({ data }: { data: Record<string, unknown> }) {
  return (
    <script
      type="application/ld+json"
      dangerouslySetInnerHTML={{
        __html: JSON.stringify(data).replace(/</g, "\\u003c"),
      }}
    />
  );
}
