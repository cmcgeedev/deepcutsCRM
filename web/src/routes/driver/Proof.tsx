import { useEffect, useRef, useState } from "react";
import SignaturePad from "signature_pad";
import type { Schemas } from "../../api/client";

type Proof = Schemas["Proof"];
const MAX = 300 * 1024;

function stripDataUrl(u: string): string {
  return u.slice(u.indexOf(",") + 1);
}

/** Downscale/recompress an image file to a JPEG under 300 KB. */
export async function shrinkImage(file: File): Promise<string> {
  const bmp = await createImageBitmap(file);
  let w = bmp.width, h = bmp.height;
  for (let pass = 0; pass < 8; pass++) {
    const c = document.createElement("canvas");
    c.width = w; c.height = h;
    const ctx = c.getContext("2d");
    if (ctx) ctx.drawImage(bmp, 0, 0, w, h);
    for (const q of [0.8, 0.6, 0.4]) {
      const url = c.toDataURL("image/jpeg", q);
      if (stripDataUrl(url).length * 3 / 4 < MAX) return stripDataUrl(url);
    }
    w = Math.round(w * 0.7); h = Math.round(h * 0.7);
  }
  throw new Error("photo could not be reduced under 300 KB");
}

export function ProofCapture({ onDone, onCancel }: { onDone: (p: Proof) => void; onCancel: () => void }) {
  const [tab, setTab] = useState<"signature" | "photo" | "name">("signature");
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const padRef = useRef<SignaturePad | null>(null);
  const [name, setName] = useState("");
  const [err, setErr] = useState<string | null>(null);

  useEffect(() => {
    if (tab !== "signature" || !canvasRef.current) return;
    const c = canvasRef.current;
    const ratio = Math.max(window.devicePixelRatio || 1, 1);
    c.width = c.offsetWidth * ratio; c.height = c.offsetHeight * ratio;
    const ctx = c.getContext("2d");
    if (ctx) ctx.scale(ratio, ratio);
    padRef.current = new SignaturePad(c, { backgroundColor: "rgb(255,255,255)" });
    return () => padRef.current?.off();
  }, [tab]);

  function doneSignature() {
    const pad = padRef.current;
    if (!pad || pad.isEmpty()) return setErr("Please sign first");
    const data = stripDataUrl(pad.toDataURL("image/png"));
    if (data.length * 3 / 4 > MAX) return setErr("Signature too large, clear and try again");
    onDone({ type: "signature", dataBase64: data });
  }

  async function donePhoto(f: File | undefined) {
    if (!f) return;
    try { onDone({ type: "photo", dataBase64: await shrinkImage(f) }); } catch (e) { setErr((e as Error).message); }
  }

  return (
    <div className="card stack">
      <div className="row">
        {(["signature", "photo", "name"] as const).map((t) => <button key={t} type="button" className={tab === t ? "" : "secondary"} onClick={() => { setTab(t); setErr(null); }}>{t}</button>)}
      </div>
      {tab === "signature" && <>
        <canvas ref={canvasRef} className="sig" aria-label="signature area" />
        <div className="row"><button type="button" className="secondary" onClick={() => padRef.current?.clear()}>Clear</button><button type="button" onClick={doneSignature}>Use signature</button></div>
      </>}
      {tab === "photo" && <label>Take a photo of the delivery<input type="file" accept="image/*" capture="environment" onChange={(e) => donePhoto(e.target.files?.[0])} /></label>}
      {tab === "name" && <>
        <label>Received by<input value={name} onChange={(e) => setName(e.target.value)} autoFocus /></label>
        <button type="button" disabled={!name.trim()} onClick={() => onDone({ type: "name", name: name.trim() })}>Use name</button>
      </>}
      {err && <p className="error">{err}</p>}
      <button type="button" className="link" onClick={onCancel}>Cancel</button>
    </div>
  );
}
