function URLName(tag) {
  return tag.Plural.replace(/\s|[#%><"]/g, "-");
}

function printf(format, ...args) {
  return format.replace(/%s/g, () => args.shift());
}

function closestTo(target, ...available) {
  // Put the target in its place inside available values
  const a = [...available, target].sort()
  // Get the two closest values to target (if target is not available, target will appear twice, so no problem here)
  const targetIdx = a.indexOf(target);
  const candidates = [a[targetIdx-1], a[targetIdx+1]].filter(c => !!c)
  if (candidates.includes(target)) {
    return target
  }
  return candidates.reverse()[0]
}

function lookupTag(name) {
  return all_tags.find((tag) =>
    [tag.Plural, tag.Singular, URLName(tag), ...(tag.Aliases || [])]
      .map((s) => s.toLowerCase())
      .includes(name)
  );
}

function lookupTech(name) {
  return all_technologies.find((tech) =>
    [tech.URLName, tech.DisplayName, ...(tech.Aliases || [])]
      .map((s) => s.toLowerCase())
      .includes(name)
  );
}

function asText(html) {
  return html.replace(/<[^>]*>/g, "");
}

function ellipsis(text, maxWords) {
	const words = text.split(" ");
	if (words.length <= maxWords) {
		return text;
	}
	return `${words.slice(0, maxWords).join(" ")}…`;
}

function Summarize(work, maxWords) {
  try {
    return (
      work.Metadata?.Summary ||
      ellipsis(asText(work.Paragraphs[0].Content), maxWords)
    );
  } catch (error) {
    throw Error(`i can't even ${JSON.stringify(work)}`);
  }
}

// TODO real ICU message format handling (right now it's just plain strings)
function translate(value) {
  // TODO ...[minify(value)] so that formatting whitespace differences doesn't prevent accessing the value
  return _translations[value] || value;
}

function AddOctothorpeIfNeeded(value) {
	if (value === "white" || value === "black") {
		return value
	}
  return value.startsWith("#") ? value : `#${value}`;
}

function ColorsMap(work) {
  let map = {};
  if (work.Metadata.Colors.Primary != "") {
    map["primary"] = AddOctothorpeIfNeeded(work.Metadata.Colors.Primary);
  }
  if (work.Metadata.Colors.Secondary != "") {
    map["secondary"] = AddOctothorpeIfNeeded(work.Metadata.Colors.Secondary);
  }
  if (work.Metadata.Colors.Tertiary != "") {
    map["tertiary"] = AddOctothorpeIfNeeded(work.Metadata.Colors.Tertiary);
  }
  return map;
}

function ColorsCSS(work) {
  return Object.entries(ColorsMap(work))
    .map(([key, value]) => `--${key}:${value}`)
    .join(";");
}

function CreatedAt(work) {
  const unparsed = work.Metadata.Created || work.Metadata.Finished;
  return unparsed ? new Date(unparsed) : new Date("0000-11-11");
}

function IsWIP(work) {
  return (
    work.Metadata.WIP ||
    (work.Metadata.Started != "" && CreatedAt(work).getFullYear() === 0)
  );
}

function ThumbnailSource(work, resolution) {
  const isLayedOutElement =
    Object.getOwnPropertyNames(work).includes("LayoutIndex");
  const key = isLayedOutElement ? work.Path : work.Media?.[0]?.Path;
  if (!key) {
    return "";
  }
  const availableResolutions = Object.keys(work.Metadata.Thumbnails[key]).map(parseFloat)
  if (!availableResolutions.length) {
  throw Error(
    `No thumbnails available for ${key}.\nAvailable thumbnails for work ${work.id}: ${Object.keys(work.Metadata.Thumbnails).join(", ")}`
  );
  }
  resolution = closestTo(resolution, ...availableResolutions)
  if (resolution > 0) {
    const thumbSource = work.Metadata.Thumbnails?.[key]?.[resolution];
    if (thumbSource) {
      return media(thumbSource.replace(/^dist\/media\//, ""));
    }
  }
  throw Error(
    `No thumbnail at size ${resolution} for ${key}.\nAvailable resolutions for ${key} (in px): ${availableResolutions.join(", ")}`
  );
}

function yearsOfWorks(works) {
  return [...new Set(works.map((w) => CreatedAt(w).getFullYear()))];
}

function withTag(works, ...tags) {
  let output = works
  for (let tag of tags) {
    if (typeof tag === "string") {
      tag = lookupTag(tag)
    }
    output = output.filter(work =>
      work.Metadata.Tags.some(t => lookupTag(t)?.Singular === tag.Singular)
    )
  }
  return output
}

function withTech(works, ...techs) {
  let output = works
  for (let tech of techs) {
    if (typeof tech === "string") {
      tech = lookupTech(tech)
    }
    output = output.filter(work =>
      work.Metadata.MadeWith.some(t => lookupTech(t)?.URLName === tech.URLName)
    )
  }
  return output
}

function withWIPStatus(status, works) {
  return works.filter((work) => IsWIP(work) === status);
}

function withCreatedYear(works, createdYear) {
  return works.filter((work) => {
    const created = CreatedAt(work);
    return created?.getFullYear() === createdYear;
  });
}

function excluding(excluseList, works) {
  return works.filter(
    (work) => !excluseList.map((w) => w.ID).includes(work.ID)
  );
}

function latestWork(works) {
  return works.sort((a, b) => CreatedAt(b) - CreatedAt(a))[0];
}

function finished(works) {
  return works.filter((w) => !IsWIP(w));
}

function unfinished(works) {
  return works.filter(IsWIP);
}

const tagged = withTag;
const madeWith = withTech;
const createdIn = withCreatedYear;

// Returns (starting row, ending row, starting column, ending column).
function PositionBounds(l) {
  let startingColumn = Number.MAX_SAFE_INTEGER;
  let startingRow = Number.MAX_SAFE_INTEGER;
  let endingColumn = 0;
  let endingRow = 0;

  // printfln("computing grid position for %s", l)
  for (let row of l.Positions) {
    if (row.length != 2) {
      throw Error(
        `A GridArea has an Indices array ${l.Positions} with a row containing ${row.length} != 2 elements`
      );
    }
    if (row[1] < startingColumn) {
      startingColumn = row[1];
    }
    if (row[0] < startingRow) {
      startingRow = row[0];
    }
    if (row[1] > endingColumn) {
      endingColumn = row[1];
    }
    if (row[0] > endingRow) {
      endingRow = row[0];
    }
  }
  return [startingRow, endingRow, startingColumn, endingColumn];
}

// CSS returns CSS statements to declare the position of that element in the content grid.
function CellCSS(l) {
  const [startingRow, endingRow, startingCol, endingCol] = PositionBounds(l);
  return `grid-row: ${startingRow + 1} / ${endingRow + 2}; grid-column: ${
    startingCol + 1
  } / ${endingCol + 2};`;
}

function IsColorBright(hexColor) {
  const hex = hexColor.replace(/^#/, "");
  const c_r = parseInt(hex.substr(0, 2), 16);
  const c_g = parseInt(hex.substr(2, 2), 16);
  const c_b = parseInt(hex.substr(4, 2), 16);
  const brightness = (c_r * 299 + c_g * 587 + c_b * 114) / 1000;
  return brightness > 155;
}
