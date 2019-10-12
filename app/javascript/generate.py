import yaml
import os

COMMON = "common"

scriptpath = os.path.dirname(os.path.realpath(__file__))

out = []
with open(os.path.join(scriptpath, "jsconfig.yml"), "r") as fin:
    cfg = yaml.safe_load(fin)

    # Append common script first
    fullpath = os.path.join(scriptpath, cfg["scriptdir"], cfg["scriptnames"]["common"])
    with open(fullpath, "r") as dep:
        out.append(dep.read())

    # Next, parse out the path of the page we're on
    out.append("var u = new URL(window.location.href);")
    out.append("var p = u.pathname;")
    out.append("var imps = [];")

    # Test if we're on a given page, and set the dependencies accordingly
    imps = []
    for page, deps in cfg["includes"].items():
        condition = " || ".join(["{}.test(p)".format(r) for r in cfg["pages"][page]])
        out.append("if ({}) {{".format(condition))
        deps = ['"' + d + '"' for d in deps]
        out.append("imps = [{}];".format(",".join(deps)))
        out.append("}")

    # Run each dependency in the order specified
    out.append("for (var importIdx = 0; importIdx < imps.length; importIdx++) {")
    out.append("switch (imps[importIdx]) {")
    for name, path in cfg["scriptnames"].items():
        if name == COMMON:
            continue
        out.append('case "{}":'.format(name))
        fullpath = os.path.join(scriptpath, cfg["scriptdir"], path)
        with open(fullpath, "r") as dep:
            out.append(dep.read())
        out.append("break;")
    out.append("}")
    out.append("}")

with open(os.path.join(scriptpath, "../static/script/main.js"), "w") as fout:
    fout.write("\n".join(out))

print("Wrote combined script")
