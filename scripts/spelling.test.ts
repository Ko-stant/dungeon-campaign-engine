// spelling:allow-file (the test cases are British on purpose)
import { describe, expect, test } from 'bun:test';
import { findBritishSpellings } from './spelling.ts';

function suggestions(text: string): string[] {
  return findBritishSpellings(text).map((f) => `${f.word}->${f.suggestion}`);
}

describe('findBritishSpellings', () => {
  test('reports the line, column and American spelling', () => {
    expect(findBritishSpellings('ok\nthe colour of rust')).toEqual([
      { line: 2, column: 5, word: 'colour', suggestion: 'color' },
    ]);
  });

  test('keeps the case of the original word', () => {
    expect(suggestions('Colour COLOUR colour')).toEqual(['Colour->Color', 'COLOUR->COLOR', 'colour->color']);
  });

  test('covers the -our, -ise, -yse, -re and doubled-l families with their inflections', () => {
    expect(
      suggestions('colourless neighbouring favourite organised realisation analysing centred travelling labelled'),
    ).toEqual([
      'colourless->colorless',
      'neighbouring->neighboring',
      'favourite->favorite',
      'organised->organized',
      'realisation->realization',
      'analysing->analyzing',
      'centred->centered',
      'travelling->traveling',
      'labelled->labeled',
    ]);
  });

  test('covers other common differences', () => {
    expect(suggestions('grey towards maths sombre plough catalogue defence judgement')).toEqual([
      'grey->gray',
      'towards->toward',
      'maths->math',
      'sombre->somber',
      'plough->plow',
      'catalogue->catalog',
      'defence->defense',
      'judgement->judgment',
    ]);
  });

  test('checks each part of camelCase and snake_case identifiers', () => {
    expect(findBritishSpellings('passageNeighbours(grey_wall)')).toEqual([
      { line: 1, column: 8, word: 'Neighbours', suggestion: 'Neighbors' },
      { line: 1, column: 19, word: 'grey', suggestion: 'gray' },
    ]);
  });

  test('ignores American spellings and look-alike words', () => {
    expect(
      suggestions(
        'color four hour tour your contour glamour rise promise otherwise advise surprise exercise ' +
          'analysis analyses emphasis synthesis dialogue Greyford axe cancellation forwards spelt constructor toString',
      ),
    ).toEqual([]);
  });

  test('skips lines marked spelling:allow', () => {
    expect(suggestions('legacy "colour" field // spelling:allow\ncolour')).toEqual(['colour->color']);
  });

  test('skips a whole text marked spelling:allow-file', () => {
    expect(suggestions('// spelling:allow-file (British word lists)\ncolour\ngrey')).toEqual([]);
  });
});
