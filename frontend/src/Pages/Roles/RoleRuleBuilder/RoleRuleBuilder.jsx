import React, { useEffect, useMemo, useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import JsonExplorer from "./JsonExplorer";
import { fetchWithAuth } from "Global/utils/Auth";
import "./RoleRuleBuilder.css";

/* =================== Backend endpoints (you'll wire these) =================== */
// Returns the current logged Pan Bagnat user (just need their login)
const CURRENT_LOGIN_API = "/api/v1/users/me"; // response: { login: "andre" } or { ft_login: "andre" }
// Returns a 42-style user payload for the given login (same shape as /v2/users/:login)
const USER_SAMPLE_API = (login) =>
  `/api/v1/admin/integrations/42/users/${encodeURIComponent(login)}`;

const ROLE_RULES_API = (roleId) =>
  `/api/v1/admin/roles/${encodeURIComponent(roleId)}/rules`;

// --- Drag guards / helpers ---
const DRAG_BLOCK_SELECTOR =
  'input, textarea, select, button, [contenteditable="true"], .rb-input, .rb-select';

function isDragBlocked(target) {
  return !!(target && target.closest && target.closest(DRAG_BLOCK_SELECTOR));
}

function setCardDragImageFromEvent(e) {
  const card = e.currentTarget.closest
    ? e.currentTarget.closest(".rb-card")
    : null;
  if (card && e.dataTransfer && typeof e.dataTransfer.setDragImage === "function") {
    const rect = card.getBoundingClientRect();
    // center-ish anchor; tweak y if you prefer
    e.dataTransfer.setDragImage(card, rect.width / 2, 20);
  }
}

function setRbDragPayload(e, rid) {
  if (!e.dataTransfer) return;
  // Use a custom type to identify our drags
  e.dataTransfer.setData("application/x-rb-rule", rid);

  // Satisfy browsers that require text, but don't provide insertable content
  e.dataTransfer.setData("text/plain", ""); // empty string → nothing gets inserted
}

function guardedDragStart(e, rule, beginDrag) {
  // If drag originates from an interactive control, block it
  if (isDragBlocked(e.target)) {
    e.preventDefault();
    e.stopPropagation();
    return;
  }
  setCardDragImageFromEvent(e);
  e.dataTransfer.setData("text/plain", rule.__rid);
  beginDrag(rule);
  e.stopPropagation();
}

function serializeRuleForSave(node) {
  if (!node) return null;
  if (node.kind === "group") {
    return {
      kind: "group",
      logic: node.logic,
      rules: (node.rules || []).map(serializeRuleForSave),
    };
  }
  if (node.kind === "array") {
    return {
      kind: "array",
      path: node.path,
      quantifier: node.quantifier,
      // normalize numeric extras
      count: node.count === "" || node.count == null ? undefined : Number(node.count),
      index: node.index === "" || node.index == null ? undefined : Number(node.index),
      predicate: serializeRuleForSave(node.predicate),
    };
  }
  if (node.kind === "scalar") {
    const out = {
      kind: "scalar",
      path: node.path,
      valueType: node.valueType,
      op: node.op,
    };
    // only include values when they matter
    if (!["exists","notExists","empty","notEmpty"].includes(node.op)) {
      out.value = node.value;
      if (node.op === "between") out.value2 = node.value2;
    }
    return out;
  }
  return null;
}

/* ----------------------- Small UI bits ----------------------- */
function Section({ title, right, children }) {
  return (
    <div className="rb-section">
      <div className="rb-section-header">
        <h3>{title}</h3>
        <div style={{ display: "flex", gap: 8, alignItems: "center" }}>{right}</div>
      </div>
      <div className="rb-section-body">{children}</div>
    </div>
  );
}
function SmallButton({ children, onClick, variant = "ghost", disabled, title }) {
  return (
    <button
      className={`rb-btn rb-btn-${variant}`}
      onClick={onClick}
      disabled={disabled}
      title={title}
    >
      {children}
    </button>
  );
}
function Modal({ open, title, children, footer, onClose }) {
  if (!open) return null;
  return (
    <div className="rb-modal-backdrop" onMouseDown={onClose}>
      <div
        className="rb-modal"
        onMouseDown={(e) => e.stopPropagation()}
        role="dialog"
        aria-modal="true"
      >
        <div className="rb-modal-header">
          <h3>{title}</h3>
        </div>
        <div className="rb-modal-body">{children}</div>
        <div className="rb-modal-footer">{footer}</div>
      </div>
    </div>
  );
}
function TextInput({
  value,
  onChange,
  placeholder,
  type = "text",
  onKeyDown,
  style,
  className = "rb-input",
}) {
  return (
    <input
      className={className}
      type={type}
      value={value ?? ""}
      placeholder={placeholder}
      onChange={(e) => onChange(e.target.value)}
      onKeyDown={onKeyDown}
      style={style}
    />
  );
}

/* ----------------------- Operators / inputs ----------------------- */
const OPS_BY_TYPE = {
  string: [
    "eq",
    "neq",
    "contains",
    "startsWith",
    "endsWith",
    "in",
    "notIn",
    "regex",
    "exists",
    "notExists",
    "empty",
    "notEmpty",
  ],
  number: [
    "eq",
    "neq",
    "gt",
    "gte",
    "lt",
    "lte",
    "between",
    "in",
    "notIn",
    "exists",
    "notExists",
  ],
  boolean: ["eq", "neq", "exists", "notExists"],
  date: ["eq", "neq", "before", "after", "between", "exists", "notExists"],
  unknown: ["exists", "notExists"],
};
function OpSelect({ valueType, value, onChange }) {
  const ops = OPS_BY_TYPE[valueType] || OPS_BY_TYPE.unknown;
  return (
    <select className="rb-select" value={value} onChange={(e) => onChange(e.target.value)}>
      {ops.map((op) => (
        <option key={op} value={op}>
          {op}
        </option>
      ))}
    </select>
  );
}
function NullableNumberInput({
  value,
  onChange,
  placeholder,
  className = "rb-input rb-input--xs",
}) {
  const v = value === "" || value == null ? "" : String(value);
  return (
    <input
      className={className}
      type="number"
      inputMode="numeric"
      step="any"
      value={v}
      placeholder={placeholder}
      onChange={(e) => {
        const t = e.target.value;
        if (t === "") onChange("");
        else onChange(Number(t));
      }}
    />
  );
}
function BooleanValueEditor({ value, onChange }) {
  return (
    <select
      className="rb-select rb-boolean"
      value={String(value)}
      onChange={(e) => onChange(e.target.value === "true")}
    >
      <option value="true">true</option>
      <option value="false">false</option>
    </select>
  );
}
function ValueInput({ valueType, value, value2, onChange }) {
  if (valueType === "boolean") {
    return <BooleanValueEditor value={!!value} onChange={(v) => onChange(v)} />;
  }
  if (valueType === "number")
    return <NullableNumberInput value={value} onChange={(v) => onChange(v)} />;
  if (valueType === "date")
    return <TextInput type="date" value={value} onChange={onChange} />;
  return <TextInput value={value} onChange={onChange} placeholder="value" />;
}

/* ----------------------- DnD infra (unchanged) ----------------------- */
let RID_SEQ = 1;
const genRid = () => `r${RID_SEQ++}`;
function withIds(rule) {
  if (!rule) return rule;
  if (rule.kind === "group")
    return {
      ...rule,
      __rid: rule.__rid || genRid(),
      rules: (rule.rules || []).map(withIds),
    };
  if (rule.kind === "array")
    return {
      ...rule,
      __rid: rule.__rid || genRid(),
      predicate: withIds(
        rule.predicate || { kind: "group", logic: "AND", rules: [] }
      ),
    };
  return { ...rule, __rid: rule.__rid || genRid() };
}
function preserveId(newRule, existingRid) {
  if (!newRule) return newRule;
  const withChildIds = withIds(newRule);
  return { ...withChildIds, __rid: existingRid };
}
function collectContainerIds(rule, set = new Set()) {
  if (!rule) return set;
  if (rule.kind === "group") {
    set.add(rule.__rid);
    (rule.rules || []).forEach((r) => collectContainerIds(r, set));
  } else if (rule.kind === "array") {
    set.add(rule.predicate.__rid);
    collectContainerIds(rule.predicate, set);
  }
  return set;
}
function findAndRemoveByRid(root, rid) {
  function walk(node) {
    if (!node) return null;
    if (node.kind === "group") {
      const arr = node.rules || [];
      const idx = arr.findIndex((r) => r.__rid === rid);
      if (idx >= 0) {
        const removed = arr[idx];
        const next = arr.slice();
        next.splice(idx, 1);
        return {
          removed,
          newNode: { ...node, rules: next },
          fromContainerRid: node.__rid,
          fromIndex: idx,
        };
      }
      for (let i = 0; i < arr.length; i++) {
        const res = walk(arr[i]);
        if (res) {
          const next = arr.slice();
          next[i] = res.newNode;
          return {
            removed: res.removed,
            newNode: { ...node, rules: next },
            fromContainerRid: res.fromContainerRid,
            fromIndex: res.fromIndex,
          };
        }
      }
      return null;
    }
    if (node.kind === "array") {
      const res = walk(node.predicate);
      if (res)
        return {
          removed: res.removed,
          newNode: { ...node, predicate: res.newNode },
          fromContainerRid: res.fromContainerRid,
          fromIndex: res.fromIndex,
        };
      return null;
    }
    return null;
  }
  const res = walk(root);
  if (!res) return null;
  return {
    removed: res.removed,
    newRoot: res.newNode,
    fromContainerRid: res.fromContainerRid,
    fromIndex: res.fromIndex,
  };
}
function insertIntoContainer(root, containerRid, index, nodeToInsert) {
  function walk(node) {
    if (node.kind === "group") {
      if (node.__rid === containerRid) {
        const arr = node.rules || [];
        const next = arr.slice();
        const safeIndex = Math.max(0, Math.min(index, next.length));
        next.splice(safeIndex, 0, nodeToInsert);
        return { newNode: { ...node, rules: next }, done: true };
      }
      const arr = node.rules || [];
      let changed = false;
      const mapped = arr.map((child) => {
        const res = walk(child);
        if (res?.done) {
          changed = true;
          return res.newNode;
        }
        return child;
      });
      return { newNode: changed ? { ...node, rules: mapped } : node, done: changed };
    }
    if (node.kind === "array") {
      const res = walk(node.predicate);
      return {
        newNode: res?.done ? { ...node, predicate: res.newNode } : node,
        done: !!res?.done,
      };
    }
    return { newNode: node, done: false };
  }
  const res = walk(root);
  return res?.newNode ?? root;
}
function DragHandle({ onDragStart, onDragEnd, title = "Drag to move" }) {
  return (
    <span
      className="rb-drag-handle"
      title={title}
      draggable
      onDragStart={onDragStart}
      onDragEnd={onDragEnd}
    >
      ⋮⋮
    </span>
  );
}
function DropZone({
  isActive,
  isForbidden,
  isOver,
  onDragEnter,
  onDragOver,
  onDragLeave,
  onDrop,
}) {
  return (
    <div
      className={[
        "rb-dropzone",
        isActive ? "rb-dropzone-active" : "",
        isOver && !isForbidden ? "rb-dropzone-over" : "",
        isForbidden && isOver ? "rb-dropzone-forbid" : "",
      ]
        .join(" ")
        .trim()}
      onDragEnter={onDragEnter}
      onDragOver={onDragOver}
      onDragLeave={onDragLeave}
      onDrop={onDrop}
    />
  );
}

/* ----------------------- Rule editor (with DnD) ----------------------- */
function RuleEditor({
  rule,
  onChange,
  onDelete,
  draggingRid,
  forbiddenContainers,
  beginDrag,
  endDrag,
  dropInto,
  containerRidForThisGroup,
  level = 0,
}) {
  if (rule.kind === "group") {
    const containerRid = rule.__rid;
    const [hoverDz, setHoverDz] = useState(null);
    const dzKey = (rid, idx) => `${rid}@${idx}`;

    const addGroup = () =>
      onChange({
        ...rule,
        rules: [
          ...rule.rules,
          withIds({ kind: "group", logic: "AND", rules: [] }),
        ],
      });
    const addScalar = () =>
      onChange({
        ...rule,
        rules: [
          ...rule.rules,
          withIds({
            kind: "scalar",
            path: "",
            op: "exists",
            value: "",
            valueType: "string",
          }),
        ],
      });
    const addArray = () =>
      onChange({
        ...rule,
        rules: [
          ...rule.rules,
          withIds({
            kind: "array",
            path: "",
            quantifier: "ANY",
            predicate: withIds({ kind: "group", logic: "AND", rules: [] }),
          }),
        ],
      });

    return (
		<div
			className="rb-card"
			draggable
			onDragStart={(e) => {guardedDragStart(e, rule, beginDrag); setRbDragPayload(e, rule.__rid);}}
			onDragEnd={endDrag}
		>
        <div className="rb-card-header">
          <div className="rb-card-title">
            <DragHandle
              onDragStart={(e) => {
				setCardDragImageFromEvent(e);
                e.dataTransfer.setData("text/plain", rule.__rid);
                beginDrag(rule);
				e.stopPropagation();
				setRbDragPayload(e, rule.__rid);
              }}
              onDragEnd={endDrag}
              title="Drag group"
            />
            <select
              className="rb-select"
              value={rule.logic}
              onChange={(e) => onChange({ ...rule, logic: e.target.value })}
            >
              <option value="AND">AND</option>
              <option value="OR">OR</option>
            </select>{" "}
            group
          </div>
          <div className="rb-card-actions">
            <SmallButton onClick={onDelete} variant="danger">
              Delete group
            </SmallButton>
          </div>
        </div>

        <div className="rb-card-body">
          <DropZone
            isActive={!!draggingRid}
            isForbidden={forbiddenContainers.has(containerRid)}
            isOver={hoverDz === dzKey(containerRid, 0)}
            onDragEnter={(e) => {
              e.preventDefault();
              setHoverDz(dzKey(containerRid, 0));
            }}
            onDragOver={(e) => e.preventDefault()}
            onDragLeave={() => setHoverDz(null)}
            onDrop={(e) => {
              e.preventDefault();
              setHoverDz(null);
              if (!forbiddenContainers.has(containerRid))
                dropInto(containerRid, 0);
            }}
          />

          {(rule.rules || []).map((child, idx) => (
            <div key={child.__rid}>
              <RuleEditor
                rule={child}
                onChange={(newChild) => {
                  const next = [...rule.rules];
                  next[idx] = preserveId(withIds(newChild), child.__rid);
                  onChange({ ...rule, rules: next });
                }}
                onDelete={() => {
                  const next = rule.rules.filter((_, i) => i !== idx);
                  onChange({ ...rule, rules: next });
                }}
                draggingRid={draggingRid}
                forbiddenContainers={forbiddenContainers}
                beginDrag={beginDrag}
                endDrag={endDrag}
                dropInto={dropInto}
                containerRidForThisGroup={containerRid}
                level={level + 1}
              />
              <DropZone
                isActive={!!draggingRid}
                isForbidden={forbiddenContainers.has(containerRid)}
                isOver={hoverDz === dzKey(containerRid, idx + 1)}
                onDragEnter={(e) => {
                  e.preventDefault();
                  setHoverDz(dzKey(containerRid, idx + 1));
                }}
                onDragOver={(e) => e.preventDefault()}
                onDragLeave={() => setHoverDz(null)}
                onDrop={(e) => {
                  e.preventDefault();
                  setHoverDz(null);
                  if (!forbiddenContainers.has(containerRid))
                    dropInto(containerRid, idx + 1);
                }}
              />
            </div>
          ))}

          <div className="rb-rule-add">
            <SmallButton onClick={addGroup}>+ Add group</SmallButton>
            <SmallButton onClick={addScalar}>+ Add scalar</SmallButton>
            <SmallButton onClick={addArray}>+ Add array</SmallButton>
          </div>
        </div>
      </div>
    );
  }

  if (rule.kind === "scalar") {
    return (
      	<div
		className="rb-card rb-card-scalar"
		draggable
		onDragStart={(e) => {guardedDragStart(e, rule, beginDrag); setRbDragPayload(e, rule.__rid);}}
		onDragEnd={endDrag}
		>
        <div className="rb-card-row">
          <DragHandle
            onDragStart={(e) => {
			setCardDragImageFromEvent(e);
              e.dataTransfer.setData("text/plain", rule.__rid);
              beginDrag(rule);
			  e.stopPropagation();
			  setRbDragPayload(e, rule.__rid);
            }}
            onDragEnd={endDrag}
            title="Drag condition"
          />
          <span className="rb-label">Path</span>
          <TextInput
            value={rule.path}
            onChange={(v) => onChange({ ...rule, path: v })}
            placeholder="e.g., project.slug"
          />
          <span className="rb-label">Type</span>
          <select
            className="rb-select"
            value={rule.valueType}
			onChange={(e) => {
				const vt = e.target.value;
				const next = {
				...rule,
				valueType: vt,
				op: defaultOpForType(vt),         // reset op for the new type
				value: defaultValueForType(vt),   // reset primary value
				value2: undefined,                // clear secondary value (e.g., between)
				};
				onChange(next);                     // this updates rootRule → re-eval happens automatically
			}}
          >
            <option value="string">string</option>
            <option value="number">number</option>
            <option value="boolean">boolean</option>
            <option value="date">date</option>
            <option value="unknown">unknown</option>
          </select>
        </div>
        <div className="rb-card-row">
          <span className="rb-label">Op</span>
          <OpSelect
            valueType={rule.valueType}
            value={rule.op}
			onChange={(op) => {
				let next = { ...rule, op };

				if (op === "between") {
				if (next.value == null || next.value === "") next.value = defaultValueForType(next.valueType);
				if (next.value2 == null || next.value2 === "") next.value2 = defaultValueForType(next.valueType);
				} else if (["exists", "notExists", "empty", "notEmpty"].includes(op)) {
				next.value = undefined;
				next.value2 = undefined;
				} else if (next.valueType === "boolean" && typeof next.value !== "boolean") {
				next.value = true; // seed sensible default
				}

				onChange(next); // triggers re-eval
			}}
          />
          {rule.op === "between" ? (
            <>
              <ValueInput
                valueType={rule.valueType}
                value={rule.value}
                onChange={(v) => onChange({ ...rule, value: v })}
              />
              <ValueInput
                valueType={rule.valueType}
                value={rule.value2}
                onChange={(v) => onChange({ ...rule, value2: v })}
              />
            </>
          ) : ["exists", "notExists", "empty", "notEmpty"].includes(rule.op) ? null : (
            <ValueInput
              valueType={rule.valueType}
              value={rule.value}
              onChange={(v) => onChange({ ...rule, value: v })}
            />
          )}
          <div className="rb-grow" />
          <SmallButton onClick={onDelete} variant="danger">
            Delete
          </SmallButton>
        </div>
      </div>
    );
  }

  if (rule.kind === "array") {
    return (
      	<div
		className="rb-card rb-card-array"
		draggable
		onDragStart={(e) => {guardedDragStart(e, rule, beginDrag); setRbDragPayload(e, rule.__rid);}}
		onDragEnd={endDrag}
		>
        <div className="rb-card-row">
          <DragHandle
            onDragStart={(e) => {
				setCardDragImageFromEvent(e);
              e.dataTransfer.setData("text/plain", rule.__rid);
              beginDrag(rule);
			  e.stopPropagation();
			  setRbDragPayload(e, rule.__rid);
            }}
            onDragEnd={endDrag}
            title="Drag array rule"
          />
          <span className="rb-label">Array Path</span>
          <TextInput
            value={rule.path}
            onChange={(v) => onChange({ ...rule, path: v })}
            placeholder="e.g., projects_users"
          />
          <span className="rb-label">Quantifier</span>
          <select
            className="rb-select"
            value={rule.quantifier}
            onChange={(e) => onChange({ ...rule, quantifier: e.target.value })}
          >
            <option value="ANY">ANY (exists)</option>
            <option value="ALL">ALL</option>
            <option value="NONE">NONE</option>
            <option value="COUNT_GTE">COUNT ≥</option>
            <option value="COUNT_EQ">COUNT =</option>
            <option value="COUNT_LTE">COUNT ≤</option>
            <option value="INDEX">INDEX i</option>
          </select>
          {["COUNT_GTE", "COUNT_EQ", "COUNT_LTE"].includes(rule.quantifier) && (
            <>
              <span className="rb-label">N</span>
              <NullableNumberInput
                value={rule.count}
                onChange={(v) => onChange({ ...rule, count: v })}
              />
            </>
          )}
          {rule.quantifier === "INDEX" && (
            <>
              <span className="rb-label">i</span>
              <NullableNumberInput
                value={rule.index}
                onChange={(v) => onChange({ ...rule, index: v })}
              />
            </>
          )}
          <div className="rb-grow" />
          <SmallButton onClick={onDelete} variant="danger">
            Delete
          </SmallButton>
        </div>

        <div className="rb-card-subtitle">Element predicate</div>
        <RuleEditor
          rule={rule.predicate}
          onChange={(pred) =>
            onChange({
              ...rule,
              predicate: preserveId(withIds(pred), rule.predicate.__rid),
            })
          }
          onDelete={() =>
            onChange({
              ...rule,
              predicate: withIds({ kind: "group", logic: "AND", rules: [] }),
            })
          }
          draggingRid={draggingRid}
          forbiddenContainers={forbiddenContainers}
          beginDrag={beginDrag}
          endDrag={endDrag}
          dropInto={dropInto}
          containerRidForThisGroup={rule.predicate.__rid}
          level={1}
        />
        <div className="rb-hint">
          Empty arrays: ANY→false, NONE→true, ALL→true. INDEX is order-dependent.
        </div>
      </div>
    );
  }

  return null;
}

/* ----------------------- Default 42 /me sample ----------------------- */
const DEFAULT_42_SAMPLE = {
  id: 2,
  email: "andre@42.fr",
  login: "andre",
  first_name: "André",
  last_name: "Aubin",
  usual_full_name: "Juliette Aubin",
  usual_first_name: "Juliette",
  url: "https://api.intra.42.fr/v2/users/andre",
  phone: null,
  displayname: "André Aubin",
  kind: "admin",
  image: {
    link: "https://cdn.intra.42.fr/users/1234567890/andre.jpg",
    versions: {
      large: "https://cdn.intra.42.fr/users/1234567890/large_andre.jpg",
      medium: "https://cdn.intra.42.fr/users/1234567890/medium_andre.jpg",
      small: "https://cdn.intra.42.fr/users/1234567890/small_andre.jpg",
      micro: "https://cdn.intra.42.fr/users/1234567890/micro_andre.jpgg",
    },
  },
  "staff?": false,
  correction_point: 4,
  pool_month: "july",
  pool_year: "2016",
  location: null,
  wallet: 0,
  anonymize_date: "2021-02-20T00:00:00.000+03:00",
  data_erasure_date: null,
  "alumni?": false,
  "active?": true,
  groups: [],
  cursus_users: [
    {
      id: 2,
      begin_at: "2017-05-14T21:37:50.172Z",
      end_at: null,
      grade: null,
      level: 0.0,
      skills: [],
      cursus_id: 1,
      has_coalition: true,
      user: {
        id: 2,
        login: "andre",
        url: "https://api.intra.42.fr/v2/users/andre",
      },
      cursus: {
        id: 1,
        created_at: "2017-11-22T13:41:00.750Z",
        name: "Piscine C",
        slug: "piscine-c",
      },
    },
  ],
  projects_users: [],
  languages_users: [
    {
      id: 2,
      language_id: 3,
      user_id: 2,
      position: 1,
      created_at: "2017-11-22T13:41:03.638Z",
    },
  ],
  achievements: [],
  titles: [],
  titles_users: [],
  partnerships: [],
  patroned: [
    {
      id: 4,
      user_id: 2,
      godfather_id: 15,
      ongoing: true,
      created_at: "2017-11-22T13:42:11.565Z",
      updated_at: "2017-11-22T13:42:11.572Z",
    },
  ],
  patroning: [],
  expertises_users: [
    {
      id: 2,
      expertise_id: 3,
      interested: false,
      value: 2,
      contact_me: false,
      created_at: "2017-11-22T13:41:22.504Z",
      user_id: 2,
    },
  ],
  roles: [],
  campus: [
    {
      id: 1,
      name: "Cluj",
      time_zone: "Europe/Bucharest",
      language: {
        id: 3,
        name: "Romanian",
        identifier: "ro",
        created_at: "2017-11-22T13:40:59.468Z",
        updated_at: "2017-11-22T13:41:26.139Z",
      },
      users_count: 28,
      vogsphere_id: 1,
    },
  ],
  campus_users: [{ id: 2, user_id: 2, campus_id: 1, is_primary: true }],
};

/* ----------------------- Page ----------------------- */
export default function RoleRuleBuilder() {
  const { roleId } = useParams();
  const navigate = useNavigate();

  // rules
  const [rootRule, setRootRule] = useState(
    withIds({ kind: "group", logic: "AND", rules: [] })
  );

  // load rules on enter
const [rulesLoading, setRulesLoading] = useState(false);
const [rulesLoadErr, setRulesLoadErr] = useState("");

async function loadRulesFromServer() {
  if (!roleId) return;
  setRulesLoading(true);
  setRulesLoadErr("");
  try {
    const res = await fetchWithAuth(ROLE_RULES_API(roleId));
    if (!res.ok) throw new Error(`HTTP ${res.status}`);
    const data = await res.json();
    const rules = data?.rules;
    // if null/undefined/empty → keep an empty root group, but note it won't pass (see eval change)
    const next = rules && typeof rules === "object"
      ? withIds(rules)
      : withIds({ kind: "group", logic: "AND", rules: [] });
    setRootRule(next);
  } catch (e) {
    setRulesLoadErr(e.message || "Failed to load rules");
  } finally {
    setRulesLoading(false);
  }
}

// on first render / when roleId changes
useEffect(() => {
  loadRulesFromServer();
  // eslint-disable-next-line react-hooks/exhaustive-deps
}, [roleId]);

  // payload source
  const [sampleSource, setSampleSource] = useState("default"); // "default" | "user"
  const [samplePayload, setSamplePayload] = useState(DEFAULT_42_SAMPLE);

  // "Pan Bagnat User" controls
  const [loginInput, setLoginInput] = useState(""); // shown in header when sampleSource === "user"
  const [loadingUser, setLoadingUser] = useState(false);
  const [loadErr, setLoadErr] = useState("");

  const [showSave, setShowSave] = useState(false);
	const [applyExisting, setApplyExisting] = useState(false);
	const [saving, setSaving] = useState(false);
	const [saveErr, setSaveErr] = useState("");
	const [saveOk, setSaveOk] = useState(false);

	useEffect(() => {
	const onKey = (e) => { if (e.key === "Escape") setShowSave(false); };
	document.addEventListener("keydown", onKey);
	return () => document.removeEventListener("keydown", onKey);
	}, []);

	const serializedRules = useMemo(() => serializeRuleForSave(rootRule), [rootRule]);

	async function applyRules() {
	setSaving(true);
	setSaveErr("");
	setSaveOk(false);
	try {
		const res = await fetchWithAuth(`/api/v1/admin/roles/${encodeURIComponent(roleId)}/rules`, {
		method: "PUT",
		headers: { "Content-Type": "application/json" },
		body: JSON.stringify({
			rules: serializedRules,
			applyToExisting: !!applyExisting,
		}),
		});
		if (!res.ok) {
		const msg = `HTTP ${res.status}`;
		try {
			const j = await res.json();
			throw new Error(j?.error || j?.message || msg);
		} catch {
			throw new Error(msg);
		}
		}
		setSaveOk(true);
		// You can close automatically after a delay if you want:
		// setTimeout(() => setShowSave(false), 800);
	} catch (e) {
		setSaveErr(e.message || "Failed to save rules");
	} finally {
		setSaving(false);
	}
	}

  useEffect(() => {
	const isRbDrag = (e) =>
		!!e.dataTransfer &&
		Array.from(e.dataTransfer.types || []).includes("application/x-rb-rule");

	const allowIfDropzone = (target) =>
		!!(target && target.closest && target.closest(".rb-dropzone"));

	const onDragOver = (e) => {
		if (isRbDrag(e) && !allowIfDropzone(e.target)) {
		e.preventDefault();           // prevent caret / text insertion feedback
		e.stopPropagation();
		if (e.dataTransfer) e.dataTransfer.dropEffect = "none";
		}
	};

	const onDrop = (e) => {
		if (isRbDrag(e) && !allowIfDropzone(e.target)) {
		e.preventDefault();           // block the actual drop outside zones
		e.stopPropagation();
		}
	};

	// Capture phase so we intercept before inputs/textareas handle it
	document.addEventListener("dragover", onDragOver, true);
	document.addEventListener("drop", onDrop, true);
	return () => {
		document.removeEventListener("dragover", onDragOver, true);
		document.removeEventListener("drop", onDrop, true);
	};
	}, []);

  // preload current logged login for convenience
  useEffect(() => {
    let cancelled = false;
    async function preloadLogin() {
      try {
        const res = await fetchWithAuth(CURRENT_LOGIN_API);
        if (!res.ok) return;
        const me = await res.json();
        const login = me?.login || me?.ft_login || me?.username || "";
        if (!cancelled && login) setLoginInput(login);
      } catch {}
    }
    preloadLogin();
    return () => {
      cancelled = true;
    };
  }, []);

  // Evaluation via backend (authoritative)
  const [evalResult, setEvalResult] = useState({ pass: false, trace: null });

  // Debounce rules before calling backend to avoid evaluating on every keystroke
  const [debouncedRules, setDebouncedRules] = useState(() => serializeRuleForSave(rootRule));
  useEffect(() => {
    const t = setTimeout(() => setDebouncedRules(serializeRuleForSave(rootRule)), 200);
    return () => clearTimeout(t);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [rootRule]);
  useEffect(() => {
    let cancelled = false;
    async function runEval() {
      // Always use backend evaluator; send current payload (default or user)
      try {
        const res = await fetchWithAuth(
          `/api/v1/admin/roles/${encodeURIComponent(roleId)}/rules/evaluate`,
          {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ rules: debouncedRules, payload: samplePayload }),
          }
        );
        if (!res) return; // fetchWithAuth already handled redirect/toast
        const data = await res.json();
        if (!cancelled) setEvalResult({ pass: !!data?.matched, trace: data?.trace || null });
      } catch (e) {
        if (!cancelled) setEvalResult({ pass: false, trace: null });
        console.error(e);
      }
    }
    runEval();
    return () => { cancelled = true; };
  }, [roleId, debouncedRules, samplePayload, sampleSource]);
  // Build coloring/messages from backend trace
  const statusByPath = useMemo(() => traceToStatusMap(evalResult.trace), [evalResult.trace]);
  const messagesByPath = useMemo(() => traceToMessageMap(evalResult.trace), [evalResult.trace]);

  function traceToStatusMap(trace) {
    const out = {};
    (function walk(t) {
      if (!t) return;
      if (t.kind === "scalar" && t.path) out[t.path] = t.result;
      if (t.kind === "array" && t.path) out[t.path] = t.result;
      if (Array.isArray(t.children)) t.children.forEach(walk);
    })(trace);
    return out;
  }

  function traceToMessageMap(trace) {
    const out = {};
    (function walk(t) {
      if (!t) return;
      if (!t.result && t.path && t.message) out[t.path] = String(t.message);
      if (Array.isArray(t.children)) t.children.forEach(walk);
    })(trace);
    return out;
  }

  // DnD
  const [draggingRid, setDraggingRid] = useState(null);
  const [forbiddenContainers, setForbiddenContainers] = useState(new Set());
  const beginDrag = (ruleNode) => {
    setDraggingRid(ruleNode.__rid);
    setForbiddenContainers(collectContainerIds(ruleNode));
  };
  const endDrag = () => {
    setDraggingRid(null);
    setForbiddenContainers(new Set());
  };
  const dropInto = (destContainerRid, destIndex) => {
    if (!draggingRid) return;
    const res = findAndRemoveByRid(rootRule, draggingRid);
    if (!res) return endDrag();
    const { removed, newRoot, fromContainerRid, fromIndex } = res;
    const forbidden = collectContainerIds(removed);
    if (forbidden.has(destContainerRid)) return endDrag();
    let insertIndex = destIndex;
    if (destContainerRid === fromContainerRid && fromIndex < destIndex)
      insertIndex = Math.max(0, destIndex - 1);
    const inserted = insertIntoContainer(newRoot, destContainerRid, insertIndex, removed);
    setRootRule(inserted);
    endDrag();
  };

  // click → add rule
  const addFromPath = (p) => {
    const rule = ruleFromPath(p, samplePayload);
    setRootRule((r) => withIds({ ...r, rules: [...r.rules, withIds(rule)] }));
  };

  // ----- payload source handlers -----
  const switchToDefault = () => {
    setSampleSource("default");
    setSamplePayload(DEFAULT_42_SAMPLE);
    setLoadErr("");
  };

  const switchToUser = async () => {
    setSampleSource("user");
    if (loginInput) await loadUserByLogin(loginInput);
  };

  const loadUserByLogin = async (login) => {
    if (!login) return;
    setLoadingUser(true);
    setLoadErr("");
    try {
      const res = await fetchWithAuth(USER_SAMPLE_API(login));
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json();
      if (!data || typeof data !== "object")
        throw new Error("Invalid JSON from backend");
      setSamplePayload(data);
    } catch (e) {
      setLoadErr(e.message || "Failed to load user");
    } finally {
      setLoadingUser(false);
    }
  };

  const onSampleChange = async (v) => {
    if (v === "default") switchToDefault();
    else if (v === "user") switchToUser();
  };

  return (
    <div className="rb-page">
      <div className="rb-header">
        <div>
          <h2>Role Rule Builder</h2>
          <div className="rb-subtitle">
            Role: <code>{roleId}</code>
          </div>
          <div className="rb-breadcrumbs">
            <span
              className="rb-link"
              onClick={() => navigate(`/admin/roles/${roleId}`)}
            >
              ← Back to role
            </span>
          </div>
        </div>
        <div className="rb-actions">
			<SmallButton onClick={loadRulesFromServer} disabled={rulesLoading}>
			{rulesLoading ? "Loading…" : "Load"}
			</SmallButton>
		  <SmallButton onClick={() => setShowSave(true)} variant="primary">Save</SmallButton>
        </div>
      </div>

      <div className="rb-grid" style={{ gridTemplateColumns: "1.2fr 1fr" }}>
        {/* Left: JSON Viewer */}
        <div className="rb-col">
          <Section
            title="JSON Viewer"
            right={
              <>
                <span
                  className={`rb-badge ${
                    evalResult.pass ? "rb-badge-pass" : "rb-badge-fail"
                  }`}
                >
                  {evalResult.pass ? "✔ Passes" : "✖ Fails"}
                </span>

                {/* Source selector */}
                <select
                  className="rb-select"
                  value={sampleSource}
                  onChange={(e) => onSampleChange(e.target.value)}
                >
                  <option value="default">42 /me (default)</option>
                  <option value="user">Pan Bagnat user</option>
                </select>

                {/* Login field + Load/Refresh (only when user source) */}
                {sampleSource === "user" && (
                  <>
                    <TextInput
                      className="rb-input"
                      value={loginInput}
                      onChange={setLoginInput}
                      placeholder="login"
                      onKeyDown={(e) => {
                        if (e.key === "Enter") loadUserByLogin(loginInput);
                      }}
                      style={{ width: 180 }}
                    />
                    <SmallButton
                      onClick={() => loadUserByLogin(loginInput)}
                      disabled={!loginInput || loadingUser}
                    >
                      {loadingUser ? "Loading…" : "Load"}
                    </SmallButton>
                    {loadErr && (
                      <span
                        className="rb-hint"
                        style={{ color: "var(--button-red)" }}
                      >
                        {loadErr}
                      </span>
                    )}
                  </>
                )}
              </>
            }
          >
            <JsonExplorer
              sample={samplePayload}
              rules={rootRule}
              diag={evalResult.trace}
              statusByPath={statusByPath}
              messagesByPath={messagesByPath}
              onAddPath={addFromPath}
              defaultExpandDepth={2}
            />
          </Section>
        </div>

        {/* Right: Conditions */}
        <div className="rb-col">
          <Section
            title="Conditions"
            right={
              <SmallButton
                onClick={() =>
                  setRootRule(
                    withIds({ kind: "group", logic: "AND", rules: [] })
                  )
                }
                variant="danger"
              >
                Clear all
              </SmallButton>
            }
          >
            <RuleEditor
              rule={rootRule}
              onChange={(nr) => setRootRule(withIds(nr))}
              onDelete={() =>
                setRootRule(withIds({ kind: "group", logic: "AND", rules: [] }))
              }
              draggingRid={draggingRid}
              forbiddenContainers={forbiddenContainers}
              beginDrag={beginDrag}
              endDrag={endDrag}
              dropInto={dropInto}
              containerRidForThisGroup={rootRule.__rid}
            />
          </Section>
        </div>
      </div>
	  <Modal
		open={showSave}
		title="Save role conditions"
		onClose={() => setShowSave(false)}
		footer={
			<>
			{saveErr && <span className="rb-hint" style={{ color: "var(--button-red)" }}>{saveErr}</span>}
			{saveOk && <span className="rb-hint" style={{ color: "var(--ok-green, #1aa34a)" }}>Saved!</span>}
			<div style={{ flex: 1 }} />
			<SmallButton onClick={() => setShowSave(false)}>Cancel</SmallButton>
			<SmallButton onClick={applyRules} variant="primary" disabled={saving}>
				{saving ? "Applying…" : "Apply"}
			</SmallButton>
			</>
		}
		>
		<p style={{ marginTop: 0 }}>
			This role will be assigned to any user validating the following conditions:
		</p>

		<div className="rb-json-preview">
			<pre><code>{JSON.stringify({ rules: serializedRules }, null, 2)}</code></pre>
		</div>

		<label className="rb-check">
			<input
			type="checkbox"
			checked={applyExisting}
			onChange={(e) => setApplyExisting(e.target.checked)}
			/>
			<span>Apply to existing users</span>
		</label>
		</Modal>
    </div>
  );
}

/* ---------- Path → rule helpers ---------- */
function ruleFromPath(path, sample) {
  const segments = path.split(".").filter(Boolean);
  const arrayIdx = segments.findIndex((seg) => seg.endsWith("[]"));
  if (arrayIdx >= 0) {
    const arrayPath = segments[arrayIdx].replace("[]", "");
    const rest = segments.slice(arrayIdx + 1).join(".");
    const emptyGroup = withIds({ kind: "group", logic: "AND", rules: [] });
    const hint = sampleElementHint(sample, arrayPath, rest);
    if (rest) emptyGroup.rules.push(withIds(makeScalarFor(rest, hint)));
    return withIds({
      kind: "array",
      path: arrayPath,
      quantifier: "ANY",
      predicate: emptyGroup,
    });
  }
  return withIds(makeScalarFor(path, safeGet(sample, path)));
}

function makeScalarFor(path, sampleValue) {
  const vt = inferValueType(sampleValue);
  const op = defaultOpForType(vt);

  // Initialize with the actual payload value when possible
  let initialValue = sampleValue;
  if (vt === "boolean") initialValue = coerceByType("boolean", sampleValue);
  else if (vt === "number") initialValue = coerceByType("number", sampleValue);
  else if (vt === "string") initialValue = coerceByType("string", sampleValue);
  else if (vt === "date") {
    // use YYYY-MM-DD if sampleValue exists, otherwise today
    const d =
      sampleValue != null
        ? new Date(sampleValue)
        : new Date();
    initialValue = isNaN(+d) ? new Date().toISOString().slice(0, 10) : d.toISOString().slice(0, 10);
  }
  if (initialValue == null && vt !== "date") {
    initialValue = defaultValueForType(vt);
  }

  return withIds({
    kind: "scalar",
    path,
    op,
    value: initialValue,
    valueType: vt,
  });
}
function sampleElementHint(sample, arrayPath, rest) {
  const arr = safeGet(sample, arrayPath);
  if (!Array.isArray(arr)) return undefined;
  const first = arr.find((x) => x != null);
  if (!first) return undefined;
  return safeGet(first, rest);
}
function inferValueType(v) {
  if (typeof v === "number") return "number";
  if (typeof v === "boolean") return "boolean";
  if (typeof v === "string" && /^\d{4}-\d{2}-\d{2}/.test(v)) return "date";
  if (typeof v === "string") return "string";
  return "unknown";
}
function defaultOpForType(t) {
  switch (t) {
    case "number":
      return "eq";
    case "boolean":
      return "eq";
    case "date":
      return "after";
    case "string":
      return "contains";
    default:
      return "exists";
  }
}
// Robust getter for dotted paths, tolerant to [] markers
function safeGet(obj, path) {
  if (!path) return obj;
  const parts = String(path).split('.').filter(Boolean);
  let cur = obj;
  for (const raw of parts) {
    const key = raw.replace(/\[\]$/, '');
    if (cur == null || typeof cur !== 'object') return undefined;
    cur = cur[key];
  }
  return cur;
}
function defaultValueForType(t) {
  switch (t) {
    case "number":
      return 0;
    case "boolean":
      return true;
    case "date":
      return new Date().toISOString().slice(0, 10);
    case "string":
      return "";
    default:
      return null;
  }
}
