import {
  documentationDirectoryPath,
  findErrorDocumentationFilesAsync,
} from "../../src/error-documentation.mjs";

async function makeRoutesAsync() {
  let paths = await findErrorDocumentationFilesAsync(
    documentationDirectoryPath
  );
  let routes = {};
  for (let path of paths) {
    routes[`/errors/${path.replace(".md", "")}/`] = "error.ejs.html";
  }
  return routes;
}

export let routes = await makeRoutesAsync();
