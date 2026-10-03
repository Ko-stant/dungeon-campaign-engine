// spelling:allow-file (the word lists below are British on purpose)
/**
 * Finds British spellings so the project stays in American English. Pure
 * logic; scripts/check-spelling.ts runs it over the repository's files.
 *
 * Words are matched whole (after splitting camelCase and snake_case), so
 * proper nouns such as "Greyford" and look-alikes such as "four" are left
 * alone. A line containing "spelling:allow" is skipped, for data that must
 * keep an outside spelling; a text containing "spelling:allow-file" is
 * skipped entirely.
 */

export interface SpellingFinding {
  line: number;
  column: number;
  word: string;
  suggestion: string;
}

/** -our words, matched anywhere in a word (colourless, unfavourable): "our" becomes "or". */
const OUR_STEMS = [
  'arbour', 'ardour', 'armour', 'behaviour', 'candour', 'clamour', 'colour', 'demeanour',
  'enamour', 'endeavour', 'favour', 'fervour', 'flavour', 'harbour', 'honour', 'humour',
  'labour', 'neighbour', 'odour', 'parlour', 'rigour', 'rumour', 'saviour', 'savour',
  'splendour', 'succour', 'tumour', 'valour', 'vapour', 'vigour',
];

/** -ise verbs, matched at the end of a word with a suffix (organised, reorganisation): "is" becomes "iz". */
const ISE_STEMS = [
  'anonymis', 'apologis', 'authoris', 'baptis', 'capitalis', 'categoris', 'centralis',
  'characteris', 'civilis', 'colonis', 'computeris', 'containeris', 'criticis', 'customis',
  'dramatis', 'emphasis', 'energis', 'equalis', 'factoris', 'familiaris', 'fertilis',
  'finalis', 'formalis', 'generalis', 'globalis', 'harmonis', 'humanis', 'hypnotis',
  'hypothesis', 'idealis', 'immunis', 'industrialis', 'initialis', 'internalis',
  'internationalis', 'italicis', 'itemis', 'jeopardis', 'legalis', 'legitimis', 'liberalis',
  'localis', 'magnetis', 'marginalis', 'materialis', 'maximis', 'mechanis', 'memoris',
  'miniaturis', 'minimis', 'mobilis', 'modernis', 'monetis', 'monopolis', 'naturalis',
  'neutralis', 'normalis', 'optimis', 'organis', 'parameteris', 'parametris', 'patronis',
  'penalis', 'personalis', 'plagiaris', 'polaris', 'popularis', 'prioritis', 'privatis',
  'publicis', 'randomis', 'rationalis', 'realis', 'recognis', 'regularis', 'revitalis',
  'sanitis', 'scrutinis', 'serialis', 'socialis', 'specialis', 'stabilis', 'standardis',
  'sterilis', 'subsidis', 'summaris', 'symbolis', 'sympathis', 'synchronis', 'synthesis',
  'systematis', 'theoris', 'tokenis', 'totalis', 'trivialis', 'urbanis', 'utilis',
  'vandalis', 'vaporis', 'vectoris', 'victimis', 'virtualis', 'visualis', 'vocalis',
];
const ISE_SUFFIXES = ['e', 'es', 'ed', 'ing', 'er', 'ers', 'ation', 'ations', 'able', 'ability'];

/** Whole words: British -> American. */
const WORDS = new Map<string, string>(Object.entries({
  // -yse
  analyse: 'analyze', analysed: 'analyzed', analysing: 'analyzing', analyser: 'analyzer',
  paralyse: 'paralyze', paralysed: 'paralyzed', paralysing: 'paralyzing',
  catalyse: 'catalyze', catalysed: 'catalyzed', catalysing: 'catalyzing',
  // -re
  centre: 'center', centres: 'centers', centred: 'centered', centring: 'centering',
  centrepiece: 'centerpiece', epicentre: 'epicenter',
  metre: 'meter', metres: 'meters', kilometre: 'kilometer', kilometres: 'kilometers',
  centimetre: 'centimeter', centimetres: 'centimeters', millimetre: 'millimeter',
  millimetres: 'millimeters', litre: 'liter', litres: 'liters',
  theatre: 'theater', theatres: 'theaters', fibre: 'fiber', fibres: 'fibers',
  calibre: 'caliber', sombre: 'somber', spectre: 'specter', spectres: 'specters',
  lustre: 'luster', meagre: 'meager', sabre: 'saber', sabres: 'sabers',
  manoeuvre: 'maneuver', manoeuvres: 'maneuvers', manoeuvred: 'maneuvered',
  manoeuvring: 'maneuvering',
  // doubled l
  travelled: 'traveled', travelling: 'traveling', traveller: 'traveler', travellers: 'travelers',
  cancelled: 'canceled', cancelling: 'canceling',
  labelled: 'labeled', labelling: 'labeling',
  modelled: 'modeled', modelling: 'modeling', modeller: 'modeler', modellers: 'modelers',
  levelled: 'leveled', levelling: 'leveling',
  fuelled: 'fueled', fuelling: 'fueling',
  signalled: 'signaled', signalling: 'signaling',
  totalled: 'totaled', totalling: 'totaling',
  marshalled: 'marshaled', marshalling: 'marshaling',
  channelled: 'channeled', channelling: 'channeling',
  tunnelled: 'tunneled', tunnelling: 'tunneling',
  quarrelled: 'quarreled', quarrelling: 'quarreling',
  dialled: 'dialed', dialling: 'dialing',
  counselling: 'counseling', counsellor: 'counselor', counsellors: 'counselors',
  rivalled: 'rivaled', equalled: 'equaled', equalling: 'equaling',
  panelled: 'paneled', panelling: 'paneling', pencilled: 'penciled', shovelled: 'shoveled',
  unravelled: 'unraveled', unravelling: 'unraveling', revelled: 'reveled',
  swivelled: 'swiveled', grovelling: 'groveling', initialled: 'initialed',
  jewellery: 'jewelry', jeweller: 'jeweler', woollen: 'woolen', libellous: 'libelous',
  // single l
  enrol: 'enroll', enrols: 'enrolls', enrolment: 'enrollment', enrolments: 'enrollments',
  fulfil: 'fulfill', fulfils: 'fulfills', fulfilment: 'fulfillment',
  instil: 'instill', instils: 'instills', distil: 'distill', distils: 'distills',
  appal: 'appall', appals: 'appalls',
  instalment: 'installment', instalments: 'installments',
  skilful: 'skillful', skilfully: 'skillfully', wilful: 'willful', wilfully: 'willfully',
  // -ence
  defence: 'defense', defences: 'defenses', offence: 'offense', offences: 'offenses',
  licence: 'license', licences: 'licenses', pretence: 'pretense',
  // -ogue
  catalogue: 'catalog', catalogues: 'catalogs', catalogued: 'cataloged',
  cataloguing: 'cataloging', analogue: 'analog', analogues: 'analogs',
  // ae / oe
  encyclopaedia: 'encyclopedia', foetus: 'fetus', oestrogen: 'estrogen',
  anaemia: 'anemia', anaemic: 'anemic', anaesthetic: 'anesthetic', anaesthesia: 'anesthesia',
  haemorrhage: 'hemorrhage', paediatric: 'pediatric', orthopaedic: 'orthopedic',
  mediaeval: 'medieval',
  // -wards
  towards: 'toward', afterwards: 'afterward', upwards: 'upward', downwards: 'downward',
  rightwards: 'rightward', leftwards: 'leftward', onwards: 'onward',
  // other
  grey: 'gray', greys: 'grays', greyed: 'grayed', greyer: 'grayer', greyest: 'grayest',
  greyish: 'grayish', greyness: 'grayness', greyscale: 'grayscale',
  programme: 'program', programmes: 'programs',
  cheque: 'check', cheques: 'checks', tyre: 'tire', tyres: 'tires', kerb: 'curb', kerbs: 'curbs',
  aluminium: 'aluminum',
  plough: 'plow', ploughs: 'plows', ploughed: 'plowed', ploughing: 'plowing',
  draught: 'draft', draughts: 'drafts',
  mould: 'mold', moulds: 'molds', mouldy: 'moldy', moulded: 'molded',
  smoulder: 'smolder', smouldering: 'smoldering',
  sceptic: 'skeptic', sceptics: 'skeptics', sceptical: 'skeptical', scepticism: 'skepticism',
  judgement: 'judgment', judgements: 'judgments',
  acknowledgement: 'acknowledgment', acknowledgements: 'acknowledgments',
  ageing: 'aging', artefact: 'artifact', artefacts: 'artifacts',
  whilst: 'while', amongst: 'among', learnt: 'learned',
  storey: 'story', storeys: 'stories', pyjamas: 'pajamas',
  cosy: 'cozy', cosier: 'cozier', cosiest: 'coziest',
  sulphur: 'sulfur', moustache: 'mustache', moustaches: 'mustaches',
  enquiry: 'inquiry', enquiries: 'inquiries', enquire: 'inquire', enquired: 'inquired',
  practise: 'practice', practised: 'practiced', practising: 'practicing',
  maths: 'math', gaol: 'jail', speciality: 'specialty', specialities: 'specialties',
}));

function americanFor(lower: string): string | undefined {
  const word = WORDS.get(lower);
  if (word !== undefined) {
    return word;
  }
  for (const stem of OUR_STEMS) {
    const at = lower.indexOf(stem);
    if (at >= 0) {
      return lower.slice(0, at) + stem.slice(0, -3) + 'or' + lower.slice(at + stem.length);
    }
  }
  for (const stem of ISE_STEMS) {
    for (const suffix of ISE_SUFFIXES) {
      if (lower.endsWith(stem + suffix)) {
        const prefix = lower.slice(0, lower.length - stem.length - suffix.length);
        return prefix + stem.slice(0, -2) + 'iz' + suffix;
      }
    }
  }
  return undefined;
}

function matchCase(original: string, american: string): string {
  if (original.length > 1 && original === original.toUpperCase()) {
    return american.toUpperCase();
  }
  const first = original.charAt(0);
  if (first !== first.toLowerCase()) {
    return american.charAt(0).toUpperCase() + american.slice(1);
  }
  return american;
}

/** Runs of letters, with camelCase split: "passageNeighbours" -> "passage", "Neighbours". */
const WORD_PATTERN = /[A-Z]+(?![a-z])|[A-Z]?[a-z]+/g;

export function findBritishSpellings(text: string): SpellingFinding[] {
  const findings: SpellingFinding[] = [];
  if (text.includes('spelling:allow-file')) {
    return findings;
  }
  text.split('\n').forEach((line, index) => {
    if (line.includes('spelling:allow')) {
      return;
    }
    for (const match of line.matchAll(WORD_PATTERN)) {
      const word = match[0];
      const american = americanFor(word.toLowerCase());
      if (american !== undefined) {
        findings.push({ line: index + 1, column: match.index + 1, word, suggestion: matchCase(word, american) });
      }
    }
  });
  return findings;
}
