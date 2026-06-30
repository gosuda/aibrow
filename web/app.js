// AIBrow web frontend.
// Handles image upload, side-by-side comparison, and parameter editing.

const imageInput = document.getElementById('image-input');
const dropZone = document.querySelector('.drop-zone');
const compareSection = document.getElementById('compare-section');
const originalImg = document.getElementById('original-img');
const editedImg = document.getElementById('edited-img');
const settingsBtn = document.getElementById('settings-btn');
const paramsSection = document.getElementById('params-section');
const paramsForm = document.getElementById('params-form');
const applyBtn = document.getElementById('apply-btn');
const actionButtons = document.getElementById('action-buttons');
const exportBtn = document.createElement('button');
exportBtn.id = 'export-btn';
exportBtn.textContent = 'Export Preset (Excel)';
actionButtons.appendChild(exportBtn);
const downloadBtn = document.createElement('button');
downloadBtn.id = 'download-btn';
downloadBtn.textContent = 'Download AI Edit';
actionButtons.appendChild(downloadBtn);

let currentImageId = null;
let currentParams = null;
let currentEditBlob = null;

const paramDefs = [
  { name: 'brightness', label: 'Brightness', min: -1, max: 1, step: 0.01, value: 0 },
  { name: 'contrast', label: 'Contrast', min: 0, max: 2, step: 0.01, value: 1 },
  { name: 'saturation', label: 'Saturation', min: 0, max: 3, step: 0.01, value: 1 },
  { name: 'gamma', label: 'Gamma', min: 0.1, max: 10, step: 0.01, value: 1 },
  { name: 'micro_contrast', label: 'Micro Contrast', min: 0, max: 3, step: 0.01, value: 0 },
  { name: 'temperature', label: 'Temperature', min: -10000, max: 10000, step: 100, value: 0 },
];

function buildParamsForm(values) {
  paramsForm.innerHTML = '';
  for (const p of paramDefs) {
    const label = document.createElement('label');
    label.textContent = p.label;
    const input = document.createElement('input');
    input.type = 'range';
    input.name = p.name;
    input.min = p.min;
    input.max = p.max;
    input.step = p.step;
    input.value = values && values[p.name] !== undefined ? values[p.name] : p.value;
    label.appendChild(input);
    paramsForm.appendChild(label);
  }
}

async function uploadFile(file) {
  const formData = new FormData();
  formData.append('image', file);

  const resp = await fetch('/api/v1/images/upload', { method: 'POST', body: formData });
  if (!resp.ok) {
    throw new Error(`Upload failed: ${resp.status}`);
  }
  const data = await resp.json();
  return data.id;
}

async function analyzeImage(id) {
  const resp = await fetch(`/api/v1/images/${id}/analyze`, { method: 'POST' });
  if (!resp.ok) {
    throw new Error(`Analyze failed: ${resp.status}`);
  }
  return resp.json();
}

async function adjustImage(id, params) {
  const resp = await fetch(`/api/v1/images/${id}/adjust`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(params),
  });
  if (!resp.ok) {
    throw new Error(`Adjust failed: ${resp.status}`);
  }
  return resp.blob();
}

async function loadImage(id, file) {
  currentImageId = id;
  currentEditBlob = null;
  originalImg.src = URL.createObjectURL(file);
  compareSection.classList.remove('hidden');
  actionButtons.classList.remove('hidden');

  try {
    const analysis = await analyzeImage(id);
    // Derive a simple starting preset from analysis.
    currentParams = {
      brightness: analysis.l_mean < 45 ? 0.1 : 0,
      contrast: analysis.l_std < 25 ? 1.15 : 1.05,
      saturation: analysis.c_mean < 15 ? 1.1 : 1.0,
      gamma: 1.0,
      micro_contrast: Math.min(1.5, analysis.micro_contrast * 100),
      temperature: 0,
    };
    await renderEdit(currentParams);
    buildParamsForm(currentParams);
  } catch (err) {
    console.error('auto analysis failed', err);
  }
}

async function renderEdit(params) {
  if (!currentImageId) return;
  const blob = await adjustImage(currentImageId, params);
  currentEditBlob = blob;
  editedImg.src = URL.createObjectURL(blob);
}

imageInput.addEventListener('change', async () => {
  const file = imageInput.files[0];
  if (!file) return;
  try {
    const id = await uploadFile(file);
    await loadImage(id, file);
  } catch (err) {
    console.error(err);
    alert(err.message);
  }
});

dropZone.addEventListener('dragover', (e) => {
  e.preventDefault();
  dropZone.classList.add('dragover');
});

dropZone.addEventListener('dragleave', () => {
  dropZone.classList.remove('dragover');
});

dropZone.addEventListener('drop', async (e) => {
  e.preventDefault();
  dropZone.classList.remove('dragover');
  const file = e.dataTransfer.files[0];
  if (!file) return;
  try {
    const id = await uploadFile(file);
    await loadImage(id, file);
  } catch (err) {
    console.error(err);
    alert(err.message);
  }
});

settingsBtn.addEventListener('click', () => {
  paramsSection.classList.toggle('hidden');
  if (!paramsSection.classList.contains('hidden')) {
    buildParamsForm(currentParams);
  }
});

applyBtn.addEventListener('click', async () => {
  if (!currentImageId) return;
  currentParams = {};
  for (const input of paramsForm.elements) {
    if (input.name) {
      currentParams[input.name] = parseFloat(input.value);
    }
  }
  try {
    await renderEdit(currentParams);
  } catch (err) {
    console.error(err);
  }
});

exportBtn.addEventListener('click', async () => {
  if (!currentImageId) return;
  const resp = await fetch(`/api/v1/images/${currentImageId}/convert?format=png`, { method: 'POST' });
  if (!resp.ok) {
    alert('Export failed');
    return;
  }
  const blob = await resp.blob();
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = `${currentImageId}_converted.png`;
  a.click();
});

downloadBtn.addEventListener('click', async () => {
  if (!currentImageId) return;
  try {
    let blob = currentEditBlob;
    if (!blob) {
      blob = await adjustImage(currentImageId, currentParams || {});
      currentEditBlob = blob;
    }
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `${currentImageId}_ai_edit.png`;
    a.click();
  } catch (err) {
    console.error(err);
    alert('Download failed');
  }
});
