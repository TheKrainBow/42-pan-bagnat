// src/Pages/Admin/JsonExplorer.jsx
import React, { useEffect, useMemo, useState } from "react";
import './JsonExplorer.css';

/**
 * Props:
 * - sample: object
 * - rules: Rule AST (for "used" highlighting)
 * - statusByPath: { [fullPath: string]: boolean }  // pass/fail per path (arrays & scalars outside arrays)
 * - diag: optional evaluator diag (still used for matched array indices)
 * - onAddPath: (pathWithMarkers: string) => void
 * - defaultExpandDepth: number (default 2)  // applies to objects; arrays start collapsed
 */
export default function JsonExplorer({
  sample,
  rules,
  statusByPath = {},
  diag,
  onAddPath,
  defaultExpandDepth = 2,
}) {
  const [expanded, setExpanded] = useState(() => new Set());

  useEffect(() => {
    const init = new Set();
    seedExpanded(sample, "", 0, defaultExpandDepth, init);
    setExpanded(init);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [defaultExpandDepth]);

  const used = useMemo(() => buildUsedPathSets(rules), [rules]);
  const matchedMap = useMemo(() => collectArrayMatches(diag), [diag]);

  const toggle = (path) => {
    setExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(path)) next.delete(path);
      else next.add(path);
      return next;
    });
  };

  return (
    <div className="rb-json">
      <JsonNode
        node={sample}
        path=""                 // FULL path of current node
        displayKey={null}       // just for rendering, not used for path computation
        isLast={true}
        expanded={expanded}
        toggle={toggle}
        onAddPath={(p) => onAddPath(normalizeToMarker(p, sample))}
        used={used}
        matchedMap={matchedMap}
        statusByPath={statusByPath}
        depth={0}
      />
    </div>
  );
}

/* ====================== Renderer ======================= */

function JsonNode({
  node,
  path,            // FULL logical path to this node
  displayKey,      // string | number | null (for pretty print)
  isLast,
  expanded,
  toggle,
  onAddPath,
  used,
  matchedMap,
  statusByPath,
  depth = 0,
}) {
  const indent = "  ".repeat(depth);

  // ARRAY
  if (Array.isArray(node)) {
    const arrPath = path; // full path of the array itself
    const isOpen = expanded.has(arrPath);
    const size = node.length;
    const usedClass = classForUsed(arrPath, used);
    const keyStatusClass = classForKeyStatus(arrPath, statusByPath);
	const successClass = classForKeySuccess(arrPath, statusByPath);

    const header = (
      <div
        className={`rb-json-line ${usedClass} ${successClass} rb-json-clickable`}
        onClick={() => onAddPath(arrPath)}
      >
        {indent}
        {displayKey != null && (
          <>
            <span
              className="rb-json-caret"
              onClick={(e) => {
                e.stopPropagation();
                toggle(arrPath);
              }}
            >
              {isOpen ? "▼" : "▶"}
            </span>
            <span className={`rb-json-key ${keyStatusClass}`}>
              "{String(displayKey)}"
            </span>
            <span className="rb-json-punct">: </span>
          </>
        )}
        <span className="rb-json-punct">[</span>
        <span className="rb-json-meta"> /* {size} */ </span>
        {!isOpen && <span className="rb-json-ellipsis"> … </span>}
        <span className="rb-json-punct">{isOpen ? "" : "]"}{isLast ? "" : ","}</span>
      </div>
    );

    if (!isOpen) return header;

    return (
      <>
        {header}
        {node.map((el, idx) => {
          const elementMatched = (matchedMap[arrPath] || []).includes(idx);
          const elementWrapClass = elementMatched ? "rb-json-match" : "";
          if (Array.isArray(el)) {
            return (
              <div key={idx} className={elementWrapClass}>
                <div className="rb-json-line">
                  {"  ".repeat(depth + 1)}
                  <span className="rb-json-punct">/* {idx} */ </span>
                </div>
                <JsonNode
                  node={el}
                  path={`${arrPath}[]`}        // element context
                  displayKey={null}
                  isLast={idx === size - 1}
                  expanded={expanded}
                  toggle={toggle}
                  onAddPath={onAddPath}
                  used={used}
                  matchedMap={matchedMap}
                  statusByPath={statusByPath}
                  depth={depth + 1}
                />
              </div>
            );
          } else if (isObject(el)) {
            return (
              <div key={idx} className={elementWrapClass}>
                <div className="rb-json-line">
                  {"  ".repeat(depth + 1)}
                  <span className="rb-json-punct">/* {idx} */ </span>
                  <span className="rb-json-punct">{"{"}</span>
                </div>
                {Object.entries(el).map(([ck, cv], i) => (
                  <JsonNode
                    key={ck}
                    node={cv}
                    path={`${arrPath}[].${ck}`} // FULL path to this field in element context
                    displayKey={ck}
                    isLast={i === Object.keys(el).length - 1}
                    expanded={expanded}
                    toggle={toggle}
                    onAddPath={onAddPath}
                    used={used}
                    matchedMap={matchedMap}
                    statusByPath={statusByPath}
                    depth={depth + 2}
                  />
                ))}
                <div className="rb-json-line">
                  {"  ".repeat(depth + 1)}
                  <span className="rb-json-punct">{"}"}</span>
                  <span className="rb-json-punct">{idx === size - 1 ? "" : ","}</span>
                </div>
              </div>
            );
          } else {
            // scalar element
            return (
              <div
                key={idx}
                className={`rb-json-line ${elementWrapClass} rb-json-clickable`}
                onClick={() => onAddPath(`${arrPath}[]`)}
              >
                {"  ".repeat(depth + 1)}
                <span className="rb-json-punct">/* {idx} */ </span>
                {renderScalar(el)}
                <span className="rb-json-punct">{idx === size - 1 ? "" : ","}</span>
              </div>
            );
          }
        })}
        <div className="rb-json-line">
          {indent}
          <span className="rb-json-punct">]</span>
          <span className="rb-json-punct">{isLast ? "" : ","}</span>
        </div>
      </>
    );
  }

  // OBJECT
  if (isObject(node)) {
    // Root object
    if (displayKey == null && path === "") {
      return (
        <>
          <div className="rb-json-line">
            <span className="rb-json-punct">{"{"}</span>
          </div>
          {Object.entries(node).map(([ck, cv], i) => (
            <JsonNode
              key={ck}
              node={cv}
              path={ck}                 // FULL path to child (no duplication)
              displayKey={ck}
              isLast={i === Object.keys(node).length - 1}
              expanded={expanded}
              toggle={toggle}
              onAddPath={onAddPath}
              used={used}
              matchedMap={matchedMap}
              statusByPath={statusByPath}
              depth={1}
            />
          ))}
          <div className="rb-json-line">
            <span className="rb-json-punct">{"}"}</span>
          </div>
        </>
      );
    }

    // Non-root object property
    const objPath = path; // already the full path for this object
    const isOpen = expanded.has(objPath);
    const usedClass = classForUsed(objPath, used);
    const successClass = classForKeySuccess(objPath, statusByPath);
    const keyStatusClass = classForKeyStatus(objPath, statusByPath);

    return (
      <>
        <div
          className={`rb-json-line ${usedClass} ${successClass} rb-json-clickable`}
          onClick={() => onAddPath(objPath)}
        >
          {indent}
          <span
            className="rb-json-caret"
            onClick={(e) => {
              e.stopPropagation();
              toggle(objPath);
            }}
          >
            {isOpen ? "▼" : "▶"}
          </span>
          <span className={`rb-json-key ${keyStatusClass}`}>"
            {String(displayKey)}"</span>
          <span className="rb-json-punct">: </span>
          <span className="rb-json-punct">{"{"}</span>
          {!isOpen && <span className="rb-json-ellipsis"> … </span>}
          <span className="rb-json-punct">{isOpen ? "" : "}"}{isLast ? "" : ","}</span>
        </div>

        {isOpen &&
          Object.entries(node).map(([ck, cv], i) => (
            <JsonNode
              key={ck}
              node={cv}
              path={`${objPath}.${ck}`}     // FULL path (no key duplication)
              displayKey={ck}
              isLast={i === Object.keys(node).length - 1}
              expanded={expanded}
              toggle={toggle}
              onAddPath={onAddPath}
              used={used}
              matchedMap={matchedMap}
              statusByPath={statusByPath}
              depth={depth + 1}
            />
          ))}

        {isOpen && (
          <div className="rb-json-line">
            {indent}
            <span className="rb-json-punct">{"}"}</span>
            <span className="rb-json-punct">{isLast ? "" : ","}</span>
          </div>
        )}
      </>
    );
  }

  // SCALAR
  const scalarPath = path; // full path to this scalar
  const usedClass = classForUsed(scalarPath, used);
  const successClass = classForKeySuccess(scalarPath, statusByPath);
  const keyStatusClass = classForKeyStatus(scalarPath, statusByPath);

  return (
    <div
      className={`rb-json-line ${usedClass} ${successClass} rb-json-clickable`}
      onClick={() => onAddPath(scalarPath)}
    >
      {indent}
      <span className={`rb-json-key ${keyStatusClass}`}>"{String(displayKey)}"</span>
      <span className="rb-json-punct">: </span>
      {renderScalar(node)}
      <span className="rb-json-punct">{isLast ? "" : ","}</span>
    </div>
  );
}

/* ====================== Helpers ======================= */

function isObject(x) {
  return x && typeof x === "object" && !Array.isArray(x);
}

function renderScalar(v) {
  if (typeof v === "string") return <span className="rb-json-string">"{v}"</span>;
  if (typeof v === "number") return <span className="rb-json-number">{v}</span>;
  if (typeof v === "boolean") return <span className="rb-json-bool">{String(v)}</span>;
  if (v === null) return <span className="rb-json-null">null</span>;
  return <span className="rb-json-string">"{String(v)}"</span>;
}

/** Expand objects up to depth; arrays are collapsed by default */
function seedExpanded(node, path, depth, maxDepth, set) {
  if (Array.isArray(node)) {
    // arrays start collapsed -> do not add their path to `set`
    const first = node.find((x) => x != null);
    if (first != null) seedExpanded(first, `${path}[]`, depth + 1, maxDepth, set);
    return;
  }
  if (isObject(node)) {
    if (depth <= maxDepth) set.add(path);
    Object.entries(node).forEach(([k, v]) => {
      const next = path ? `${path}.${k}` : k;
      seedExpanded(v, next, depth + 1, maxDepth, set);
    });
  }
}

/** Mark arrays with [] in path where relevant */
function normalizeToMarker(path, sample) {
  const parts = path.split(".");
  let cur = sample;
  const out = [];
  for (const seg of parts) {
    if (!seg) continue;
    const key = seg.replace(/\[\]$/, "");
    const val = cur?.[key];
    if (Array.isArray(val)) {
      out.push(`${key}[]`);
      const first = val.find((x) => x != null) ?? {};
      cur = first;
    } else {
      out.push(key);
      cur = val;
    }
  }
  return out.join(".");
}

/** Used highlighting sets */
function buildUsedPathSets(root) {
  const used = new Set();
  if (root) collectUsed(root, "", used);
  const ancestors = new Set();
  for (const p of used) {
    const parts = p.split(".");
    for (let i = 1; i < parts.length; i++) ancestors.add(parts.slice(0, i).join("."));
  }
  return { used, ancestors };
}
function collectUsed(rule, base, set) {
  if (!rule) return;
  if (rule.kind === "group") return (rule.rules || []).forEach((r) => collectUsed(r, base, set));
  if (rule.kind === "scalar") return set.add(join(base, rule.path));
  if (rule.kind === "array") {
    const arrPath = join(base, rule.path);
    set.add(arrPath);
    collectUsed(rule.predicate, `${arrPath}[]`, set);
  }
}
function join(a, b) { if (!a) return b || ""; if (!b) return a; return `${a}.${b}`; }

function classForUsed(path, { used, ancestors }) {
  if (!path) return "";
  if (used.has(path) || used.has(`${path}[]`)) return "rb-json-used";
  if (ancestors.has(path)) return "rb-json-ancestor";
  return "";
}

function classForKeySuccess(path, statusByPath) {
  if (path == null || path === "") return "";
  const v = statusByPath[path];
  if (v === true) return "rb-json-pass";
  if (v === false) return "rb-json-fail";
  return "";
}

function classForKeyStatus(path, statusByPath) {
  if (path in statusByPath) return statusByPath[path] ? "rb-json-key-pass" : "rb-json-key-fail";
  return "";
}

function collectArrayMatches(diag) {
  const map = {};
  (function walk(d) {
    if (!d || typeof d !== "object") return;
    if (d.kind === "group" && Array.isArray(d.children)) d.children.forEach((c) => walk(c.diag || c));
    if (d.kind === "array" && Array.isArray(d.matchedIndices)) map[d.path] = (map[d.path] || []).concat(d.matchedIndices);
    if (Array.isArray(d.children) && d.kind !== "group") d.children.forEach((c) => walk(c.diag || c));
  })(diag);
  return map;
}
